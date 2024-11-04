package hub

import (
	"log"
	"sync"

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

// Hub ...
type Hub struct {
	sync.RWMutex
	options     Options
	mqttClient  *mqtt.ClientConn
	device      *mtrf.Connection
	writeRouter *router.Router
}

// New создает инстанс Hub и подключается к брокеру
// Возвращает ошибку если не получилось подключиться
func New(device *mtrf.Connection, options Options) (*Hub, error) {
	h := &Hub{
		options:     options,
		device:      device,
		writeRouter: router.New(),
	}

	// register routes
	h.init()

	go h.mqttLoop()
	go h.deviceWorker()

	return h, nil
}

// Publish отправляет сообщение брокеру
func (h *Hub) Publish(topic string, payload string) {
	h.RLock()
	mc := h.mqttClient
	h.RUnlock()

	if mc == nil {
		return
	}

	log.Printf("[mqtt] <- %s: %s", h.options.Topic+"/"+topic, payload)
	mc.Publish(&proto.Publish{
		Header:    proto.Header{},
		TopicName: h.options.Topic + "/" + topic,
		Payload:   proto.BytesPayload([]byte(payload)),
	})
}

func (h *Hub) deviceWorker() {
	for {
		r := <-h.device.Recv()
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
