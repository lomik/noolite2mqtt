package hub

import (
	"testing"

	"github.com/lomik/noolite2mqtt/pkg/mtrf"
	"github.com/stretchr/testify/assert"
)

func TestExpandResponse(t *testing.T) {
	assert := assert.New(t)

	table := map[string](map[string]string){
		"[173,2,0,0,7,130,0,2,0,1,255,0,0,203,182,187,174]": {
			"txf/7/0000CBB6/state/bind":       "off",
			"txf/7/0000CBB6/state/brightness": "255",
			"txf/7/0000CBB6/state/power":      "on",
		},
		"[173,1,0,16,42,21,7,205,32,48,255,0,0,0,0,32,174]": {
			"rx/42/temperature": "20.5",
			"rx/42/humidity":    "48",
			"rx/42/low_battery": "false",
			"rx/42/device":      "PT111",
		},
		"[173,1,0,7,44,0,0,0,0,0,0,0,0,0,0,225,174]": {
			"rx/44/power": "off",
		},
		"[173,1,0,8,44,2,0,0,0,0,0,0,0,0,0,228,174]": {
			"rx/44/power": "on",
		},
	}

	for body, expected := range table {
		response, err := mtrf.JSONResponse([]byte(body))
		assert.NoError(err)
		assert.Equal(expected, expandResponse(response))
	}
}

func TestFormatTemperature(t *testing.T) {
	assert := assert.New(t)

	table := []struct {
		d0, d1   uint8
		expected string
	}{
		{0x00, 0x00, "0.0"},
		{205, 0x00, "20.5"},    // 205
		{205, 0x20, "20.5"},    // биты типа датчика не влияют
		{205, 0x80, "20.5"},    // бит батареи не влияет
		{0xFB, 0x0F, "-0.5"},   // 0xFFB = -5
		{0xFF, 0x0F, "-0.1"},   // 0xFFF = -1
		{0x9C, 0x0F, "-10.0"},  // 0xF9C = -100
		{0x5F, 0x0F, "-16.1"},  // 0xF5F = -161
		{0x5F, 0xAF, "-16.1"},  // тип датчика и батарея не влияют
		{0xFF, 0x07, "204.7"},  // 0x7FF - максимум
		{0x00, 0x08, "-204.8"}, // 0x800 - минимум
	}

	for _, c := range table {
		assert.Equal(c.expected, formatTemperature(c.d0, c.d1), "D0=%#x D1=%#x", c.d0, c.d1)
	}
}

func TestResponseCodes(t *testing.T) {
	assert := assert.New(t)

	// привязка выполнена
	r := &mtrf.Response{Mode: mtrf.ModeTXF, Ctr: mtrf.RespBindDone, Ch: 7, ID2: 0xCB, ID3: 0xB6}
	assert.Equal(map[string]string{"txf/7/0000CBB6/bind": "done"}, expandResponse(r))
	assert.NoError(responseError(r))

	r = &mtrf.Response{Mode: mtrf.ModeRX, Ctr: mtrf.RespBindDone, Ch: 44}
	assert.Equal(map[string]string{"rx/44/bind": "done"}, expandResponse(r))

	// нет ответа от блока: топиков нет, есть ошибка
	r = &mtrf.Response{Mode: mtrf.ModeTXF, Ctr: mtrf.RespNoResponse, Ch: 7, Cmd: mtrf.CmdSendState}
	assert.Empty(expandResponse(r))
	assert.EqualError(responseError(r), "txf/7/00000000: no response from block")

	r = &mtrf.Response{Mode: mtrf.ModeTX, Ctr: mtrf.RespError, Ch: 3}
	assert.Empty(expandResponse(r))
	assert.EqualError(responseError(r), "tx/3: error during execution")

	r = &mtrf.Response{Mode: mtrf.ModeService, Ctr: mtrf.RespError}
	assert.EqualError(responseError(r), "mode 4 ch 0: error during execution")

	// ответ сервисного режима не разворачивается в топики
	r = &mtrf.Response{Mode: mtrf.ModeService, ID1: 1, ID2: 103, ID3: 170}
	assert.Empty(expandResponse(r))
	assert.NoError(responseError(r))
}
