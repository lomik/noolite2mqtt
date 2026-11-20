package router

import (
	"fmt"
	"strings"
)

type entry struct {
	callback    func(ctx interface{})
	staticChild map[string]*entry
	paramChild  map[string]*entry
}

// Router ...
type Router struct {
	root   *entry
	params map[string](func(value string, ctx interface{}) error)
}

// New creates new instance of Router
func New() *Router {
	return &Router{
		root:   newEntry(),
		params: make(map[string](func(value string, ctx interface{}) error)),
	}
}

func newEntry() *entry {
	return &entry{
		staticChild: make(map[string]*entry),
		paramChild:  make(map[string]*entry),
	}
}

// AddPath ...
func (r *Router) AddPath(path string, callback func(ctx interface{})) {
	a := strings.Split(path, "/")
	p := r.root
	for _, w := range a {
		if strings.HasPrefix(w, ":") {
			n := p.paramChild[w[1:]]
			if n == nil {
				n = newEntry()
				p.paramChild[w[1:]] = n
			}
			p = n
		} else {
			n := p.staticChild[w]
			if n == nil {
				n = newEntry()
				p.staticChild[w] = n
			}
			p = n
		}
	}

	p.callback = callback
}

// AddParam ...
func (r *Router) AddParam(name string, parser func(value string, ctx interface{}) error) {
	r.params[name] = parser
}

func (r *Router) lookup(root *entry, path []string) (*entry, map[string]string) {
	// is leaf. Узел без callback - промежуточный (например, "tx" для
	// "tx/:ch/on"), совпадением он не считается, чтобы поиск мог
	// продолжиться по другим веткам
	if len(path) == 0 {
		if root.callback == nil {
			return nil, nil
		}
		return root, make(map[string]string)
	}

	// check static
	if c, exists := root.staticChild[path[0]]; exists {
		e, m := r.lookup(c, path[1:])
		if e != nil {
			return e, m
		}
	}

	for k, v := range root.paramChild {
		e, m := r.lookup(v, path[1:])
		if e != nil {
			m[k] = path[0]
			return e, m
		}
	}

	return nil, nil
}

// Route ...
func (r *Router) Route(path string, ctx interface{}) error {
	e, m := r.lookup(r.root, strings.Split(path, "/"))
	if e == nil || e.callback == nil {
		return fmt.Errorf("no route for %#v", path)
	}

	for p, v := range m {
		parser := r.params[p]
		if parser != nil {
			if err := parser(v, ctx); err != nil {
				return err
			}
		}
	}

	e.callback(ctx)

	return nil
}
