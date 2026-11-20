package hub

import (
	"sync/atomic"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

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

	// Таймаут установки соединения с брокером (TCP + CONNECT/CONNACK).
	// 0 - значение по умолчанию
	ConnectTimeout time.Duration
	// Интервал MQTT keepalive (PINGREQ). Должен быть не меньше секунды.
	// 0 - значение по умолчанию
	KeepAlive time.Duration
	// Сколько ждать PINGRESP, прежде чем считать соединение потерянным.
	// 0 - значение по умолчанию
	PingTimeout time.Duration
}

type writeEvent struct {
	topicName string
	ctx       writeContext
}

// Hub ...
type Hub struct {
	lastRecv    atomic.Int64 // время последнего ответа модуля, UnixNano
	probes      atomic.Int32 // количество отправленных проб, ответ на которые еще не пришел
	options     Options
	mqttClient  mqtt.Client
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
	// отсчет для watchdog ведется от момента запуска
	h.lastRecv.Store(time.Now().UnixNano())

	// register routes
	h.init()

	h.mqttClient = h.newMQTTClient()
	// с ConnectRetry первая попытка и все повторы идут в фоне,
	// поэтому ждать токен не нужно
	h.mqttClient.Connect()

	go h.deviceWorker()
	go h.writeWorker()

	return h, nil
}

func (h *Hub) deviceWorker() {
	for {
		r := <-h.device.Recv()
		h.lastRecv.Store(time.Now().UnixNano())
		if r.Mode == mtrf.ModeService && h.takeProbe() {
			// ответ на пробу watchdog, в MQTT не публикуем
			continue
		}
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
	return time.Unix(0, h.lastRecv.Load())
}
