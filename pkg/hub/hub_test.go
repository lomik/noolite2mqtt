package hub

import (
	"io"
	"net"
	"sync"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/assert"
)

// fakeBroker принимает TCP-соединения и сообщает о каждом в канал accepted.
// Если connack == true, на первый пакет (CONNECT) отвечает CONNACK,
// после чего молча вычитывает все входящие данные и ничего не отвечает
// (в том числе на PINGREQ и SUBSCRIBE). Иначе не отвечает вообще.
type fakeBroker struct {
	ln       net.Listener
	accepted chan net.Conn
}

func newFakeBroker(t *testing.T, connack bool) *fakeBroker {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	b := &fakeBroker{ln: ln, accepted: make(chan net.Conn, 16)}
	t.Cleanup(func() { ln.Close() })

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			t.Cleanup(func() { conn.Close() })
			b.accepted <- conn
			go func() {
				if connack {
					buf := make([]byte, 1024)
					if _, err := conn.Read(buf); err != nil {
						return
					}
					// CONNACK: session present = 0, return code = 0
					if _, err := conn.Write([]byte{0x20, 0x02, 0x00, 0x00}); err != nil {
						return
					}
				}
				io.Copy(io.Discard, conn)
			}()
		}
	}()
	return b
}

func (b *fakeBroker) addr() string {
	return b.ln.Addr().String()
}

func (b *fakeBroker) waitConn(t *testing.T, timeout time.Duration) {
	t.Helper()
	select {
	case <-b.accepted:
	case <-time.After(timeout):
		t.Fatal("no connection to broker")
	}
}

func newTestHub(t *testing.T, options Options) *Hub {
	if options.Topic == "" {
		options.Topic = "test"
	}
	if options.ClientID == "" {
		options.ClientID = "test"
	}
	h := &Hub{options: options, writeChan: make(chan writeEvent, 100)}
	h.mqttClient = h.newMQTTClient()
	h.mqttClient.Connect()
	t.Cleanup(func() { h.mqttClient.Disconnect(0) })
	return h
}

func TestBrokerURL(t *testing.T) {
	assert := assert.New(t)
	assert.Equal("tcp://127.0.0.1:1883", brokerURL("127.0.0.1:1883"))
	assert.Equal("tcp://broker:1883", brokerURL("tcp://broker:1883"))
	assert.Equal("ssl://broker:8883", brokerURL("ssl://broker:8883"))
}

// Брокер принял TCP-соединение, но не отвечает на CONNECT.
// Раньше подключение висело на этом вечно, теперь должно
// оборваться по таймауту и повториться
func TestConnectTimeoutRetries(t *testing.T) {
	b := newFakeBroker(t, false)
	newTestHub(t, Options{Broker: b.addr(), ConnectTimeout: 300 * time.Millisecond})

	b.waitConn(t, 5*time.Second)
	b.waitConn(t, 5*time.Second)
}

// Брокер ответил на CONNECT и замолчал (полуоткрытое соединение).
// Keepalive должен обнаружить это и инициировать переподключение
func TestKeepAliveDetectsDeadBroker(t *testing.T) {
	b := newFakeBroker(t, true)
	h := newTestHub(t, Options{
		Broker:      b.addr(),
		KeepAlive:   time.Second,
		PingTimeout: 500 * time.Millisecond,
	})

	b.waitConn(t, 5*time.Second)
	b.waitConn(t, 10*time.Second)
	_ = h
}

// Без брокера Publish не должен ни паниковать, ни блокироваться
func TestPublishWithoutBroker(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close() // никто не слушает

	h := newTestHub(t, Options{Broker: addr})

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 100; i++ {
			h.Publish("recv/raw", "[]")
		}
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Publish blocked without broker")
	}
}

// Publish вызывается из нескольких горутин. Под -race тест ловит гонки
func TestConcurrentPublish(t *testing.T) {
	b := newFakeBroker(t, true)
	h := newTestHub(t, Options{Broker: b.addr()})
	b.waitConn(t, 5*time.Second)

	var wg sync.WaitGroup
	for g := 0; g < 4; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				h.Publish("recv/raw", "[]")
			}
		}()
	}
	wg.Wait()
}

type testMessage struct {
	topic   string
	payload []byte
}

func (m *testMessage) Duplicate() bool   { return false }
func (m *testMessage) Qos() byte         { return 0 }
func (m *testMessage) Retained() bool    { return false }
func (m *testMessage) Topic() string     { return m.topic }
func (m *testMessage) MessageID() uint16 { return 0 }
func (m *testMessage) Payload() []byte   { return m.payload }
func (m *testMessage) Ack()              {}

var _ mqtt.Message = &testMessage{}

// Входящая команда превращается в writeEvent без префикса topic/write/,
// а при переполненной очереди отбрасывается, не блокируя клиент paho
func TestOnMQTTMessage(t *testing.T) {
	assert := assert.New(t)

	h := &Hub{options: Options{Topic: "test"}, writeChan: make(chan writeEvent, 1)}

	h.onMQTTMessage(nil, &testMessage{topic: "test/write/txf/15/on", payload: []byte("1")})

	done := make(chan struct{})
	go func() {
		defer close(done)
		h.onMQTTMessage(nil, &testMessage{topic: "test/write/txf/15/off"})
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("onMQTTMessage blocked on full queue")
	}

	assert.Len(h.writeChan, 1)
	w := <-h.writeChan
	assert.Equal("txf/15/on", w.topicName)
	assert.Equal("1", w.ctx.payload)
}
