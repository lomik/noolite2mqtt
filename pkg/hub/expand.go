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

func expandResponse(r *mtrf.Response) (topicPayload map[string]string) {
	topicPayload = make(map[string]string)

	if r.Ctr != 0 {
		return
	}

	set := func(topic string, value interface{}) {
		if r.Mode == mtrf.ModeTXF {
			topicPayload[fmt.Sprintf("txf/%d/%s/%s", r.Ch, r.Device(), topic)] = toString(value)
		}
		if r.Mode == mtrf.ModeTX {
			topicPayload[fmt.Sprintf("tx/%d/%s", r.Ch, topic)] = toString(value)
		}
		if r.Mode == mtrf.ModeRXF {
			topicPayload[fmt.Sprintf("rxf/%d/%s/%s", r.Ch, r.Device(), topic)] = toString(value)
		}
		if r.Mode == mtrf.ModeRX {
			topicPayload[fmt.Sprintf("rx/%d/%s", r.Ch, topic)] = toString(value)
		}
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
		t := int(r.D0) + (int(r.D1&0xf) << 8)
		d := sensorDevice[(r.D1&0x70)>>4]
		if d == "" {
			d = "unknown"
		}
		set("temperature", fmt.Sprintf("%d.%d", t/10, t%10))
		set("humidity", r.D2)
		set("low_battery", stateBoolean[r.D1>>7])
		set("device", d)
	}

	return
}
