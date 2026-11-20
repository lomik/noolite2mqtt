package router

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLookup(t *testing.T) {
	assert := assert.New(t)
	r := New()

	addPath := func(path string) {
		r.AddPath(path, func(ctx interface{}) { *(ctx.(*string)) = path })
	}

	test := func(path string, expectedRoute string, expectedParams map[string]string) {
		e, m := r.lookup(r.root, strings.Split(path, "/"))
		if expectedRoute == "" {
			assert.Nil(e, path)
			assert.Nil(m, path)
		} else {
			var p string
			if assert.NotNil(e, path) {
				e.callback(&p)
				assert.Equal(expectedRoute, p, path)
				assert.Equal(expectedParams, m, path)
			}
		}
	}

	addPath("rx/:ch/on")
	addPath("raw")

	test("rx/42/on", "rx/:ch/on", map[string]string{"ch": "42"})
	test("rx/42/off", "", nil)
	test("raw", "raw", map[string]string{})

	// промежуточный узел не считается маршрутом
	test("rx/42", "", nil)
	test("rx", "", nil)
}

// Статическая ветка совпала по префиксу, но не ведет к маршруту:
// поиск должен откатиться на параметрическую ветку
func TestLookupFallbackToParam(t *testing.T) {
	assert := assert.New(t)
	r := New()

	var got string
	r.AddPath("a/static/x/y", func(ctx interface{}) { got = "static" })
	r.AddPath("a/:p/x", func(ctx interface{}) { got = "param" })

	assert.NoError(r.Route("a/static/x", nil))
	assert.Equal("param", got)

	assert.NoError(r.Route("a/static/x/y", nil))
	assert.Equal("static", got)

	assert.Error(r.Route("a/static", nil))
}
