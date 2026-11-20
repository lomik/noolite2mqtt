package hub

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	proto "github.com/huin/mqtt"
	"github.com/jeffallen/mqtt"

	"github.com/lomik/noolite2mqtt/pkg/mtrf"
	"github.com/lomik/noolite2mqtt/pkg/router"
)

// Options ...
type Options struct {
	Broker   string // MQTT broken
	Topic    string // MQTT root topic
	ClientID string // MQTT client ID
	User     string // MQTT user
	Password string // MQTT password
}

type writeEvent struct {
	topicName string
	ctx       writeContext
}

// Hub ...
type Hub struct {
	sync.RWMutex
	lastRecv    atomic.Int64
	options     Options
	mqttClient  *mqtt.ClientConn
	device      *mtrf.Connection
	writeRouter *router.Router
	writeChan   chan writeEvent
}

// New создает инстанс Hub и подключается к брокеру
// Возвращает ошибку если не получилось подключиться
func New(device *mtrf.Connection, options Options) (*Hub, error) {
	h := &Hub{
		options:     options,
		device:      device,
		writeRouter: router.New(),
		writeChan:   make(chan writeEvent, 100),
	}

	// register routes
	h.init()

	go h.mqttLoop()
	go h.deviceWorker()
	go h.writeWorker()

	return h, nil
}

// Publish отправляет сообщение брокеру.
// Если соединения с брокером нет, сообщение отбрасывается.
func (h *Hub) Publish(topic string, payload string) {
	// ClientConn не потокобезопасен, поэтому все обращения к нему
	// сериализуются под эксклюзивной блокировкой
	h.Lock()
	defer h.Unlock()

	if h.mqttClient == nil {
		return
	}

	log.Printf("[mqtt] <- %s: %s", h.options.Topic+"/"+topic, payload)
	err := safePublish(h.mqttClient, &proto.Publish{
		Header:    proto.Header{},
		TopicName: h.options.Topic + "/" + topic,
		Payload:   proto.BytesPayload([]byte(payload)),
	})
	if err != nil {
		log.Printf("[mqtt] publish failed, connection is dead: %s", err.Error())
		h.mqttClient = nil
	}
}

// safePublish вызывает mc.Publish и перехватывает панику.
// При обрыве соединения библиотека закрывает внутренний канал отправки,
// и Publish паникует с "send on closed channel". Узнать о закрытии канала
// заранее нельзя, поэтому единственный надежный способ - recover.
func safePublish(mc *mqtt.ClientConn, m *proto.Publish) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	mc.Publish(m)
	return nil
}

func (h *Hub) deviceWorker() {
	for {
		r := <-h.device.Recv()
		h.lastRecv.Store(time.Now().Unix())
		h.Publish("recv/raw", r.JSON())
		for k, v := range expandResponse(r) {
			h.Publish("recv/"+k, v)
		}
	}
}

func (h *Hub) onError(err error) {
	h.Publish("error", err.Error())
}

func (h *Hub) sendRequest(r *mtrf.Request) {
	h.device.Send() <- r
	h.Publish("sent/raw", r.JSON())
}

// Loop ... . @TODO: выходить когда порвалась связь с брокером или модулем
func (h *Hub) Loop() error {
	select {}
}

func (h *Hub) LastRecv() time.Time {
	return time.Unix(h.lastRecv.Load(), 0)
}
