package hub

import (
	"fmt"

	"github.com/lomik/noolite2mqtt/pkg/mtrf"
)

var statePower = map[uint8]string{
	0: "off",
	1: "on",
	2: "temporary_on",
}

var stateOnOff = map[uint8]string{
	0: "off",
	1: "on",
}

var stateBoolean = map[uint8]string{
	0: "false",
	1: "true",
}

var sensorDevice = map[uint8]string{
	1: "PT112",
	2: "PT111",
}

func toString(i interface{}) string {
	switch v := i.(type) {
	case uint8:
		return fmt.Sprintf("%d", v)
	case int:
		return fmt.Sprintf("%d", v)
	case string:
		return v
	default:
		panic("not implemented")
	}
}

// formatTemperature декодирует температуру датчика из D0 и D1.
// Температура - 12-битное число со знаком (дополнительный код) в десятых
// долях градуса: D0 - младшие 8 бит, биты 3..0 байта D1 - старшие 4 бита.
// Остальные биты D1 (тип датчика и разряд батареи) игнорируются
func formatTemperature(d0, d1 uint8) string {
	t := int(d0) | int(d1&0x0f)<<8
	if t&0x800 != 0 { // знаковый бит
		t -= 0x1000
	}

	sign := ""
	if t < 0 {
		sign = "-"
		t = -t
	}
	return fmt.Sprintf("%s%d.%d", sign, t/10, t%10)
}

// responsePrefix возвращает начало топика для ответа модуля:
// режим, канал и (для nooLite-F) адрес устройства.
// Для остальных режимов (сервисный, обновление ПО) возвращает ""
func responsePrefix(r *mtrf.Response) string {
	switch r.Mode {
	case mtrf.ModeTX:
		return fmt.Sprintf("tx/%d", r.Ch)
	case mtrf.ModeRX:
		return fmt.Sprintf("rx/%d", r.Ch)
	case mtrf.ModeTXF:
		return fmt.Sprintf("txf/%d/%s", r.Ch, r.Device())
	case mtrf.ModeRXF:
		return fmt.Sprintf("rxf/%d/%s", r.Ch, r.Device())
	default:
		return ""
	}
}

// responseError возвращает ошибку, если модуль сообщил о неудачном
// выполнении команды (код ответа в поле Ctr)
func responseError(r *mtrf.Response) error {
	where := responsePrefix(r)
	if where == "" {
		where = fmt.Sprintf("mode %d ch %d", r.Mode, r.Ch)
	}

	switch r.Ctr {
	case mtrf.RespNoResponse:
		return fmt.Errorf("%s: no response from block", where)
	case mtrf.RespError:
		return fmt.Errorf("%s: error during execution", where)
	}
	return nil
}

func expandResponse(r *mtrf.Response) (topicPayload map[string]string) {
	topicPayload = make(map[string]string)

	prefix := responsePrefix(r)
	if prefix == "" {
		return
	}

	set := func(topic string, value interface{}) {
		topicPayload[prefix+"/"+topic] = toString(value)
	}

	switch r.Ctr {
	case mtrf.RespDone:
		// разбираем команду ниже
	case mtrf.RespBindDone:
		set("bind", "done")
		return
	default:
		// ошибки обрабатываются в responseError
		return
	}

	switch r.Cmd {
	case mtrf.CmdOff:
		set("power", "off")
	case mtrf.CmdOn:
		set("power", "on")
	case mtrf.CmdSwitch:
		set("switch", "")
	case mtrf.CmdLoadPreset:
		set("load_preset", "")
	case mtrf.CmdSavePreset:
		set("save_preset", "")
	case mtrf.CmdSendState:
		switch r.Fmt {
		case 0:
			set("state/power", statePower[r.D2&0xf])
			set("state/bind", stateOnOff[r.D2>>7])
			set("state/brightness", r.D3)
		case 1:
			set("state/input", stateOnOff[r.D2])
			set("state/noolite_disabled_temporary", stateBoolean[(r.D3>>1)&0x1])
			set("state/noolite_disabled", stateBoolean[r.D3&0x1])
		case 2:
			set("state/free_slots_noolite", r.D2)
			set("state/free_slots_noolite_f", r.D3)
		}
	case mtrf.CmdBrightBack:
		set("bright_back", "")
	case mtrf.CmdStopReg:
		set("stop_reg", "")
	case mtrf.CmdSensTempHumi:
		d := sensorDevice[(r.D1&0x70)>>4]
		if d == "" {
			d = "unknown"
		}
		set("temperature", formatTemperature(r.D0, r.D1))
		set("humidity", r.D2)
		set("low_battery", stateBoolean[r.D1>>7])
		set("device", d)
	}

	return
}
