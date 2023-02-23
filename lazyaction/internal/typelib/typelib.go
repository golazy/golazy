package typelib

import "reflect"

type TypeLib struct {
	lib map[string]reflect.Value
}

func New() *TypeLib {
	return &TypeLib{
		lib: make(map[string]reflect.Value),
	}
}

func (t *TypeLib) Register(name string, value interface{}) {
	t.lib[name] = reflect.ValueOf(value)
}

func (t *TypeLib) Get(name string) (interface{}, bool) {
	v, ok := t.lib[name]
	if !ok {
		return nil, false
	}
	return v.Interface(), true
}
