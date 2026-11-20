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
			"rx/42/sensor/temperature": "20.5",
			"rx/42/sensor/humidity":    "48",
			"rx/42/sensor/low_battery": "false",
			"rx/42/sensor/device":      "PT111",
		},
		"[173,1,0,7,44,0,0,0,0,0,0,0,0,0,0,225,174]": {
			"rx/44/off": "",
		},
		"[173,1,0,8,44,2,0,0,0,0,0,0,0,0,0,228,174]": {
			"rx/44/on": "",
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
