package mtrf

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJsonRequest(t *testing.T) {
	assert := assert.New(t)

	func() {
		r, err := JSONRequest([]byte("[171,1,0,26,44,2,0,0,0,0,0,0,0,0,0,246,172]"))
		assert.NoError(err)
		assert.Equal(&Request{St: 171, Mode: 1, Ctr: 0, Res: 26, Ch: 44, Cmd: 2, Fmt: 0, D0: 0, D1: 0, D2: 0, D3: 0, ID0: 0, ID1: 0, ID2: 0, ID3: 0, Crc: 246, Sp: 172}, r)
	}()

	badMessage := []string{
		"[171,1,26,44,2,0,0,0,0,0,0,0,0,0,246,172]",    // короткое
		"[171,1,0,-1,44,2,0,0,0,0,0,0,0,0,0,246,172]",  // отрицательное число
		"[171,1,0,256,44,2,0,0,0,0,0,0,0,0,0,246,172]", // большое число
	}

	for _, m := range badMessage {
		func() {
			r, err := JSONRequest([]byte(m))
			assert.Error(err)
			assert.Nil(r)
		}()
	}
}

func TestRequestSerializationDoesNotMutate(t *testing.T) {
	assert := assert.New(t)

	r := &Request{Mode: ModeTXF, Ch: 11, Cmd: CmdOff}
	orig := *r

	assert.Equal("[171,2,0,0,11,0,0,0,0,0,0,0,0,0,0,184,172]", r.JSON())
	assert.Contains(r.String(), "St: 171")
	assert.Contains(r.String(), "Crc: 184")
	assert.Equal([]byte{171, 2, 0, 0, 11, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 184, 172}, r.Bytes())

	// сериализация не должна менять сам Request
	assert.Equal(orig, *r)
}

// Request отправляется в модуль и одновременно логируется в MQTT.
// Под -race этот тест ловит гонку, если сериализация пишет в поля
func TestRequestConcurrentSerialization(t *testing.T) {
	r := &Request{Mode: ModeTXF, Ch: 11, Cmd: CmdOn}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 1000; i++ {
			r.Bytes()
		}
	}()
	for i := 0; i < 1000; i++ {
		_ = r.JSON()
		_ = r.String()
	}
	<-done
}

func TestServiceRequest(t *testing.T) {
	assert.Equal(t,
		[]byte{171, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 175, 172},
		ServiceRequest().Bytes(),
	)
}
