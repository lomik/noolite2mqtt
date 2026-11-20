package hub

import (
	"net"
	"testing"
	"time"

	"github.com/jeffallen/mqtt"
	"github.com/stretchr/testify/assert"
)

// После обрыва соединения с брокером Publish не должен паниковать,
// а мертвый клиент должен сбрасываться
func TestPublishAfterBrokerDisconnect(t *testing.T) {
	assert := assert.New(t)

	client, server := net.Pipe()
	cc := mqtt.NewClientConn(client)

	h := &Hub{options: Options{Topic: "test"}, mqttClient: cc}

	// Брокер рвет соединение: reader библиотеки получает EOF
	// и закрывает внутренний канал отправки и Incoming
	server.Close()

	select {
	case _, ok := <-cc.Incoming:
		assert.False(ok)
	case <-time.After(5 * time.Second):
		t.Fatal("Incoming was not closed after disconnect")
	}

	assert.NotPanics(func() {
		h.Publish("recv/raw", "[]")
	})

	h.RLock()
	assert.Nil(h.mqttClient)
	h.RUnlock()

	// Повторные вызовы без клиента тоже безопасны
	assert.NotPanics(func() {
		h.Publish("recv/raw", "[]")
	})
}
