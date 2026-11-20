package hub

import (
	"log"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const (
	defaultConnectTimeout = 10 * time.Second
	defaultKeepAlive      = 30 * time.Second
	defaultPingTimeout    = 10 * time.Second
	// сколько максимум ждать отправки одного сообщения брокеру
	publishTimeout = 10 * time.Second
)

// brokerURL добавляет схему tcp:// к адресу вида host:port,
// чтобы сохранить совместимость со старым форматом флага -server
func brokerURL(broker string) string {
	if strings.Contains(broker, "://") {
		return broker
	}
	return "tcp://" + broker
}

func (h *Hub) newMQTTClient() mqtt.Client {
	o := h.options

	connectTimeout := o.ConnectTimeout
	if connectTimeout <= 0 {
		connectTimeout = defaultConnectTimeout
	}
	keepAlive := o.KeepAlive
	if keepAlive <= 0 {
		keepAlive = defaultKeepAlive
	}
	pingTimeout := o.PingTimeout
	if pingTimeout <= 0 {
		pingTimeout = defaultPingTimeout
	}

	opts := mqtt.NewClientOptions().
		AddBroker(brokerURL(o.Broker)).
		SetClientID(o.ClientID).
		SetUsername(o.User).
		SetPassword(o.Password).
		SetCleanSession(true).
		SetConnectTimeout(connectTimeout).
		SetKeepAlive(keepAlive).
		SetPingTimeout(pingTimeout).
		SetWriteTimeout(publishTimeout).
		// повторять первое подключение и переподключаться после обрыва
		SetConnectRetry(true).
		SetConnectRetryInterval(time.Second).
		SetAutoReconnect(true).
		SetMaxReconnectInterval(10 * time.Second).
		// порядок команд важен (on/off), поэтому обработчик вызывается
		// последовательно и не должен блокироваться
		SetOrderMatters(true).
		SetOnConnectHandler(h.onMQTTConnect).
		SetConnectionLostHandler(func(_ mqtt.Client, err error) {
			log.Printf("[mqtt] connection lost: %s", err.Error())
		}).
		SetReconnectingHandler(func(_ mqtt.Client, _ *mqtt.ClientOptions) {
			log.Printf("[mqtt] reconnecting to %s", o.Broker)
		})

	return mqtt.NewClient(opts)
}

// onMQTTConnect вызывается после каждого (пере)подключения.
// Сессия чистая, поэтому подписку нужно оформлять заново каждый раз
func (h *Hub) onMQTTConnect(c mqtt.Client) {
	log.Printf("connected to broker %s with client id %#v", h.options.Broker, h.options.ClientID)

	topic := h.options.Topic + "/write/#"
	t := c.Subscribe(topic, 0, h.onMQTTMessage)
	if !t.WaitTimeout(publishTimeout) {
		log.Printf("[mqtt] subscribe to %s timed out", topic)
		return
	}
	if err := t.Error(); err != nil {
		log.Printf("[mqtt] subscribe to %s failed: %s", topic, err.Error())
	}
}

// onMQTTMessage обрабатывает входящую команду.
// Вызывается из горутины paho и не должен блокироваться, иначе встанет
// обработка входящих пакетов клиента. Если очередь команд переполнена
// (например, модуль MTRF недоступен), команда отбрасывается
func (h *Hub) onMQTTMessage(_ mqtt.Client, m mqtt.Message) {
	payload := string(m.Payload())
	log.Printf("[mqtt] -> %s: %s", m.Topic(), payload)

	topicName := strings.TrimPrefix(m.Topic(), h.options.Topic+"/write/")

	select {
	case h.writeChan <- writeEvent{
		topicName: topicName,
		ctx:       writeContext{payload: payload},
	}:
	default:
		log.Printf("[mqtt] write queue is full, dropped %s", m.Topic())
	}
}

// Publish отправляет сообщение брокеру (QoS 0).
// Если соединения с брокером нет, сообщение отбрасывается.
// Клиент paho потокобезопасен, дополнительная синхронизация не нужна.
// Вызов блокируется не дольше publishTimeout
func (h *Hub) Publish(topic string, payload string) {
	topic = h.options.Topic + "/" + topic
	log.Printf("[mqtt] <- %s: %s", topic, payload)

	t := h.mqttClient.Publish(topic, 0, false, payload)
	// для QoS 0 ошибка (например, нет соединения) известна сразу,
	// ждать подтверждения не нужно
	select {
	case <-t.Done():
		if err := t.Error(); err != nil {
			log.Printf("[mqtt] publish to %s failed: %s", topic, err.Error())
		}
	default:
	}
}
