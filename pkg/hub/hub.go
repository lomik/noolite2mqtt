package hub

import (
	"bytes"
	"log"
	"net"
	"strings"

	proto "github.com/huin/mqtt"
	"github.com/jeffallen/mqtt"

	"github.com/lomik/nooLiteHub/pkg/mtrf"
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
	options    Options
	mqttConn   net.Conn
	mqttClient *mqtt.ClientConn
	device     *mtrf.Connection
}

// New создает инстанс Hub и подключается к брокеру
// Возвращает ошибку если не получилось подключиться
func New(device *mtrf.Connection, options Options) (*Hub, error) {
	h := &Hub{
		options: options,
		device:  device,
	}

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

// ждет новые события из mqtt
func (h *Hub) mqttWorker() {
	for m := range h.mqttClient.Incoming {
		b := new(bytes.Buffer)
		m.Payload.WritePayload(b)
		log.Printf("[mqtt] -> %s: %s", m.TopicName, b.String())

		topicName := m.TopicName
		topicName = strings.TrimPrefix(topicName, h.options.Topic+"/write/")
		h.handleWrite(topicName, b.String())
	}
}

func (h *Hub) deviceWorker() {
	for {
		r := <-h.device.Recv()
		h.Publish("in/raw", r.JSON())
	}
}

func (h *Hub) onError(err error) {
	h.Publish("error", err.Error())
}

// Обработчик сообщений от mqtt
func (h *Hub) handleWrite(topic string, payload string) {
	if topic == "raw" {
		r, err := mtrf.JSONRequest([]byte(payload))
		if err != nil {
			h.onError(err)
			return
		}
		h.device.Send() <- r
	}
}

// Loop ... . @TODO: выходить когда порвалась связь с брокером или модулем
func (h *Hub) Loop() error {
	select {}
}
