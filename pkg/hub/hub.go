package hub

import (
	"log"
	"net"

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
	options     Options
	mqttConn    net.Conn
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

	// подключиться к порту брокера
	mqttConn, err := net.Dial("tcp", h.options.Broker)
	if err != nil {
		return nil, err
	}

	cc := mqtt.NewClientConn(mqttConn)
	cc.Dump = false
	cc.ClientId = h.options.ClientID

	tq := make([]proto.TopicQos, 1)
	tq[0].Topic = h.options.Topic + "/write/#"
	tq[0].Qos = proto.QosAtMostOnce

	if err := cc.Connect(h.options.User, h.options.User); err != nil {
		mqttConn.Close()
		return nil, err
	}
	log.Printf("connected to broker %s with client id %#v", h.options.Broker, cc.ClientId)
	cc.Subscribe(tq)

	h.mqttConn = mqttConn
	h.mqttClient = cc

	go h.mqttWorker()
	go h.deviceWorker()

	return h, nil
}

// Publish отправляет сообщение брокеру
func (h *Hub) Publish(topic string, payload string) {
	log.Printf("[mqtt] <- %s: %s", h.options.Topic+"/"+topic, payload)
	h.mqttClient.Publish(&proto.Publish{
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
