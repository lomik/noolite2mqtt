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
