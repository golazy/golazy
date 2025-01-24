package lazycontroller

import (
	"fmt"
	"reflect"

	"golazy.dev/lazysupport"
)

func (c *Base) DisableAutoRender() {
	c.noAutoRender = true
}

func (c *Base) ViewVar(name string, val any) {
	c.viewVars[name] = val
}
func (c *Base) ViewSplat(val any) {
	for k, v := range objToMap(val) {
		c.viewVars[k] = v
	}
}
func (c *Base) ViewSet(value any) {
	t := reflect.TypeOf(value)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Struct:
		c.viewVars[t.Name()] = value
	case reflect.Slice:
		sliceType := t.Elem()
		for sliceType.Kind() == reflect.Ptr {
			sliceType = sliceType.Elem()
		}
		c.viewVars[lazysupport.Pluralize(sliceType.Name())] = value
	default:
		panic(fmt.Sprintf("when rendering views, found data that is not a struct or slice: %T", value))
	}
}
