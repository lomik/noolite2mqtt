package hub

import (
	"io"
	"net"
	"testing"
	"time"

	"github.com/lomik/noolite2mqtt/pkg/mtrf"
	"github.com/stretchr/testify/assert"
)

// ответ реального модуля MTRF-64 на запрос сервисного режима
var serviceResponse = []byte{173, 4, 0, 0, 0, 0, 0, 0, 3, 1, 0, 0, 1, 103, 170, 199, 174}

// newFakeModule поднимает TCP-сервер, имитирующий модуль MTRF-64.
// Если alive == true, на каждый запрос сервисного режима отвечает
// serviceResponse, иначе молча вычитывает запросы
func newFakeModule(t *testing.T, alive bool) string {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			t.Cleanup(func() { conn.Close() })
			go func() {
				buf := make([]byte, mtrf.MessageLen)
				for {
					if _, err := io.ReadFull(conn, buf); err != nil {
						return
					}
					if alive && buf[1] == mtrf.ModeService {
						if _, err := conn.Write(serviceResponse); err != nil {
							return
						}
					}
				}
			}()
		}
	}()
	return ln.Addr().String()
}

func newTestHubWithModule(t *testing.T, alive bool) *Hub {
	// брокера нет: Publish просто отбрасывает сообщения
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	broker := ln.Addr().String()
	ln.Close()

	h := newTestHub(t, Options{Broker: broker})
	h.device = mtrf.Connect(newFakeModule(t, alive))
	t.Cleanup(h.device.Close)
	h.lastRecv.Store(time.Now().UnixNano())
	go h.deviceWorker()
	return h
}

func TestTakeProbe(t *testing.T) {
	assert := assert.New(t)

	h := &Hub{}
	assert.False(h.takeProbe())

	h.probes.Add(2)
	assert.True(h.takeProbe())
	assert.True(h.takeProbe())
	assert.False(h.takeProbe())
	assert.Equal(int32(0), h.probes.Load())
}

// Модуль молчит в эфире, но отвечает на пробы: watchdog не срабатывает
func TestWatchdogAliveModule(t *testing.T) {
	h := newTestHubWithModule(t, true)

	failed := make(chan struct{})
	go h.Watchdog(1500*time.Millisecond, func() { close(failed) })

	select {
	case <-failed:
		t.Fatal("watchdog fired for alive module")
	case <-time.After(5 * time.Second):
	}

	// ответы на пробы засчитаны и не копятся
	assert.LessOrEqual(t, h.probes.Load(), int32(1))
	assert.WithinDuration(t, time.Now(), h.LastRecv(), 1500*time.Millisecond)
}

// Модуль не отвечает: watchdog срабатывает
func TestWatchdogDeadModule(t *testing.T) {
	h := newTestHubWithModule(t, false)

	failed := make(chan struct{})
	go h.Watchdog(1500*time.Millisecond, func() { close(failed) })

	select {
	case <-failed:
	case <-time.After(5 * time.Second):
		t.Fatal("watchdog did not fire for dead module")
	}
}
