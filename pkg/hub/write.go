package hub

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	proto "github.com/huin/mqtt"
	"github.com/jeffallen/mqtt"
	"github.com/lomik/noolite2mqtt/pkg/mtrf"
)

type writeContext struct {
	ch      uint8
	id0     uint8
	id1     uint8
	id2     uint8
	id3     uint8
	payload string
}

func (h *Hub) init() {
	h.writeRouter.AddParam("ch", func(value string, ctx interface{}) error {
		i, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		if i < 0 || i > 63 {
			return fmt.Errorf("ch value %d out of range [0, 63]", i)
		}
		ctx.(*writeContext).ch = uint8(i)
		return nil
	})

	h.writeRouter.AddParam("device", func(value string, ctx interface{}) error {
		if len(value) != 8 {
			return fmt.Errorf("invalid length of device id, expected 8")
		}

		v, err := strconv.ParseInt(value, 16, 64)
		if err != nil {
			return err
		}

		ctx.(*writeContext).id0 = uint8((v >> 24) % 256)
		ctx.(*writeContext).id1 = uint8((v >> 16) % 256)
		ctx.(*writeContext).id2 = uint8((v >> 8) % 256)
		ctx.(*writeContext).id3 = uint8(v % 256)
		return nil
	})

	h.write("raw", func(ctx *writeContext) {
		r, err := mtrf.JSONRequest([]byte(ctx.payload))
		if err != nil {
			h.onError(err)
			return
		}
		h.sendRequest(r)
	})

	// TX topics
	h.write("tx/:ch/power", func(ctx *writeContext) {
		if ctx.payload == "on" || ctx.payload == "true" {
			h.sendRequest(&mtrf.Request{Mode: mtrf.ModeTX, Ch: ctx.ch, Cmd: mtrf.CmdOn})
		} else {
			h.sendRequest(&mtrf.Request{Mode: mtrf.ModeTX, Ch: ctx.ch, Cmd: mtrf.CmdOff})
		}
	})

	h.write("tx/:ch/on", func(ctx *writeContext) {
		h.sendRequest(&mtrf.Request{Mode: mtrf.ModeTX, Ch: ctx.ch, Cmd: mtrf.CmdOn})
	})

	h.write("tx/:ch/off", func(ctx *writeContext) {
		h.sendRequest(&mtrf.Request{Mode: mtrf.ModeTX, Ch: ctx.ch, Cmd: mtrf.CmdOff})
	})

	h.write("tx/:ch/switch", func(ctx *writeContext) {
		h.sendRequest(&mtrf.Request{Mode: mtrf.ModeTX, Ch: ctx.ch, Cmd: mtrf.CmdSwitch})
	})

	h.write("tx/:ch/bind", func(ctx *writeContext) {
		h.sendRequest(&mtrf.Request{Mode: mtrf.ModeTX, Ch: ctx.ch, Cmd: mtrf.CmdBind})
	})

	h.write("tx/:ch/unbind", func(ctx *writeContext) {
		h.sendRequest(&mtrf.Request{Mode: mtrf.ModeTX, Ch: ctx.ch, Cmd: mtrf.CmdUnbind})
	})

	// TX-F topics
	h.write("txf/:ch/power", func(ctx *writeContext) {
		if ctx.payload == "on" || ctx.payload == "true" {
			h.sendRequest(&mtrf.Request{Mode: mtrf.ModeTXF, Ch: ctx.ch, Cmd: mtrf.CmdOn})
		} else {
			h.sendRequest(&mtrf.Request{Mode: mtrf.ModeTXF, Ch: ctx.ch, Cmd: mtrf.CmdOff})
		}
	})

	h.write("txf/:ch/on", func(ctx *writeContext) {
		h.sendRequest(&mtrf.Request{Mode: mtrf.ModeTXF, Ch: ctx.ch, Cmd: mtrf.CmdOn})
	})

	h.write("txf/:ch/off", func(ctx *writeContext) {
		h.sendRequest(&mtrf.Request{Mode: mtrf.ModeTXF, Ch: ctx.ch, Cmd: mtrf.CmdOff})
	})

	h.write("txf/:ch/switch", func(ctx *writeContext) {
		h.sendRequest(&mtrf.Request{Mode: mtrf.ModeTXF, Ch: ctx.ch, Cmd: mtrf.CmdSwitch})
	})

	h.write("txf/:ch/bind", func(ctx *writeContext) {
		h.sendRequest(&mtrf.Request{Mode: mtrf.ModeTXF, Ch: ctx.ch, Cmd: mtrf.CmdBind})
	})

	h.write("txf/:ch/unbind", func(ctx *writeContext) {
		h.sendRequest(&mtrf.Request{Mode: mtrf.ModeTXF, Ch: ctx.ch, Cmd: mtrf.CmdUnbind})
	})

	h.write("txf/:ch/state", func(ctx *writeContext) {
		h.sendRequest(&mtrf.Request{Mode: mtrf.ModeTXF, Ch: ctx.ch, Cmd: mtrf.CmdReadState})
	})

	h.write("txf/:ch/state0", func(ctx *writeContext) {
		h.sendRequest(&mtrf.Request{Mode: mtrf.ModeTXF, Ch: ctx.ch, Cmd: mtrf.CmdReadState})
	})

	h.write("txf/:ch/state1", func(ctx *writeContext) {
		h.sendRequest(&mtrf.Request{Mode: mtrf.ModeTXF, Ch: ctx.ch, Cmd: mtrf.CmdReadState, Fmt: 1})
	})

	h.write("txf/:ch/state2", func(ctx *writeContext) {
		h.sendRequest(&mtrf.Request{Mode: mtrf.ModeTXF, Ch: ctx.ch, Cmd: mtrf.CmdReadState, Fmt: 2})
	})

	// RX
	h.write("rx/:ch/bind", func(ctx *writeContext) {
		h.sendRequest(&mtrf.Request{Mode: mtrf.ModeRX, Ch: ctx.ch, Ctr: mtrf.CtrBindOn})
	})

	// RX-F
	h.write("rxf/:ch/bind", func(ctx *writeContext) {
		h.sendRequest(&mtrf.Request{Mode: mtrf.ModeRXF, Ch: ctx.ch, Ctr: mtrf.CtrBindOn})
	})

	// h.write("txf/:ch/:device/read_state", func(ctx *writeContext) {
	// 	h.sendRequest(&mtrf.Request{Mode: mtrf.ModeTXF, Ctr: mtrf.CtrSendF, Ch: ctx.ch, ID0: ctx.id0, ID1: ctx.id1, ID2: ctx.id2, ID3: ctx.id3, Cmd: mtrf.CmdReadState})
	// })
}

// регистрирует callback на входящее mqtt сообщение
func (h *Hub) write(path string, callback func(ctx *writeContext)) {
	h.writeRouter.AddPath(path, func(ctx interface{}) {
		callback(ctx.(*writeContext))
	})
}

func (h *Hub) mqttLoop() {
	for {
		err := h.mqttWorker()
		if err != nil {
			log.Printf("mqtt worker failed: %s", err.Error())
		} else {
			log.Printf("mqtt loop exited without error")
		}
		time.Sleep(time.Second)
	}
}

// ждет новые события из mqtt
func (h *Hub) mqttWorker() error {
	// подключиться к порту брокера
	mqttConn, err := net.Dial("tcp", h.options.Broker)
	if err != nil {
		return err
	}

	cc := mqtt.NewClientConn(mqttConn)
	cc.Dump = false
	cc.ClientId = h.options.ClientID

	tq := make([]proto.TopicQos, 1)
	tq[0].Topic = h.options.Topic + "/write/#"
	tq[0].Qos = proto.QosAtMostOnce

	if err := cc.Connect(h.options.User, h.options.User); err != nil {
		mqttConn.Close()
		return err
	}
	log.Printf("connected to broker %s with client id %#v", h.options.Broker, cc.ClientId)
	cc.Subscribe(tq)

	h.Lock()
	h.mqttClient = cc
	h.Unlock()

	for m := range h.mqttClient.Incoming {
		b := new(bytes.Buffer)
		m.Payload.WritePayload(b)
		log.Printf("[mqtt] -> %s: %s", m.TopicName, b.String())

		topicName := m.TopicName
		topicName = strings.TrimPrefix(topicName, h.options.Topic+"/write/")

		ctx := &writeContext{payload: b.String()}
		if err := h.writeRouter.Route(topicName, ctx); err != nil {
			return err
		}
	}

	return nil
}
