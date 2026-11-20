package hub

import (
	"time"

	"github.com/lomik/noolite2mqtt/pkg/mtrf"
)

// Probe отправляет модулю запрос сервисного режима (MODE=4), на который
// модуль отвечает сам, без передачи в эфир. Ответ обновляет LastRecv и
// не публикуется в MQTT.
// Не блокируется: если очередь отправки переполнена (например, модуль
// недоступен), проба не отправляется и возвращается false
func (h *Hub) Probe() bool {
	// счетчик увеличивается до отправки, иначе ответ может прийти раньше
	h.probes.Add(1)
	select {
	case h.device.Send() <- mtrf.ServiceRequest():
		return true
	default:
		h.probes.Add(-1)
		return false
	}
}

// takeProbe засчитывает ответ сервисного режима как ответ на пробу.
// Возвращает false, если проб в ожидании нет: тогда это ответ на запрос
// пользователя (например, через write/raw) или на запрос при подключении,
// и его нужно опубликовать как обычно
func (h *Hub) takeProbe() bool {
	for {
		n := h.probes.Load()
		if n <= 0 {
			return false
		}
		if h.probes.CompareAndSwap(n, n-1) {
			return true
		}
	}
}

// Watchdog каждые timeout/3 отправляет модулю пробу и вызывает onFail,
// если от модуля ничего не приходило дольше timeout. После вызова onFail
// завершается. Блокирует вызывающую горутину
func (h *Hub) Watchdog(timeout time.Duration, onFail func()) {
	t := time.NewTicker(timeout / 3)
	defer t.Stop()

	for range t.C {
		if time.Since(h.LastRecv()) > timeout {
			onFail()
			return
		}
		h.Probe()
	}
}
