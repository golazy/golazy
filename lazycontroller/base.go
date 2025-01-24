package lazycontroller

import (
	"context"
	"fmt"
	"net/http"
	"reflect"

	"github.com/gorilla/sessions"
	"golazy.dev/lazycontext"
	"golazy.dev/lazydispatch"
	"golazy.dev/lazyview"
	"golazy.dev/lazyview/head"
)

type Base struct {
	R     *http.Request
	W     http.ResponseWriter
	Route *lazydispatch.Route
	Views *lazyview.Views

	// Render and views
	viewVars     map[string]interface{}
	bodySent     bool
	noAutoRender bool
	Layout       string

	// Session
	session           *sessions.Session
	sessionHasChanges bool

	// CSRF
	csrf string

	// params
	params Params
	head.Head
}

func (c *Base) Before_000_Init(ctx context.Context, r *http.Request, w http.ResponseWriter, route *lazydispatch.Route) {
	c.R = r
	c.W = w
	c.Route = route
	c.Views = lazycontext.Get[*lazyview.Views](ctx)
	c.viewVars = map[string]any{
		"Controller": route.Controller,
		"Action":     route.Action,
	}

}

func (c *Base) After_ZZZ_Close() {
	if !c.bodySent {
		c.Render(nil)
	}
}

func objToMap(data any) map[string]any {
	vars := map[string]any{}
	t := reflect.TypeOf(data)
	v := reflect.ValueOf(data)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
		v = v.Elem()
	}
	switch t.Kind() {
	case reflect.Struct:
		for _, field := range reflect.VisibleFields(t) {
			if field.IsExported() {
				vars[field.Name] = v.Field(field.Index[0]).Interface()
			}
		}
	case reflect.Map:
		if t.Key().Kind() != reflect.String {
			panic(fmt.Sprintf("data must be a map with string keys, not %s", t.Key().Kind()))
		}
		for _, key := range v.MapKeys() {
			vars[key.String()] = v.MapIndex(key).Interface()
		}
	default:
		panic(fmt.Sprintf("data must be a struct or map, not %s", t.Kind()))
	}

	return vars
}
