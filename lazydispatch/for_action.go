// pacakge actionhandler creates an http handler for a given controller/method
package lazydispatch

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"sync"
)

type methodInfo struct {
	method int
	name   string
	nIn    int
	nOut   int
}

func genMethodInfo(m reflect.Method) methodInfo {
	return methodInfo{method: m.Index, name: m.Name, nIn: m.Type.NumIn(), nOut: m.Type.NumOut()}
}

func forAction[T any](controller T, action string, ctxfn ...func(ctx context.Context, r *http.Request) context.Context) http.Handler {
	// Validate input
	tt := reflect.TypeOf(controller)
	if tt.Kind() != reflect.Ptr {
		panic("controller must be a pointer to a struct")
	}
	if tt.Elem().Kind() != reflect.Struct {
		panic("controller must be a pointer to a struct")
	}
	tt = reflect.TypeOf(controller).Elem()
	vv := reflect.ValueOf(controller).Elem()

	actx := &actionctx{}
	actx.befores = make([]methodInfo, 0)
	actx.afters = make([]methodInfo, 0)
	actx.generators = make(map[string]methodInfo)
	actx.t = reflect.TypeOf(controller)
	actx.tt = tt
	actx.vv = vv
	actx.ctxfn = ctxfn

	// Setup instance pool
	actx.pool = sync.Pool{
		New: func() any {
			instance := reflect.New(tt)
			instance.Elem().Set(vv)
			return &instance
		},
	}

	// Fill action
	actionm, ok := actx.t.MethodByName(action)
	if !ok {
		panic(fmt.Sprintf("action %s not found in controller %s", action, actx.t.String()))
	}
	actx.action = genMethodInfo(actionm)

	// Fill generators and filters
	m := actx.t.NumMethod()
	for i := 0; i < m; i++ {
		m := actx.t.Method(i)
		mi := genMethodInfo(m)
		switch {
		case strings.HasPrefix(mi.name, "Before_") || strings.HasPrefix(mi.name, "BEFORE"):
			actx.befores = append(actx.befores, mi)
		case strings.HasPrefix(mi.name, "After_") || strings.HasPrefix(mi.name, "AFTER"):
			actx.afters = append(actx.afters, mi)
		case mi.name == "HandleError":
			actx.ErrorHandler = &mi
		case strings.HasPrefix(mi.name, "Gen_") || strings.HasPrefix(mi.name, "GEN"):
			if mi.nOut == 0 {
				panic(fmt.Sprintf("Gen functions must return 1 or 2 values: %s", mi.name))
			}

			actx.generators[genType(m)] = mi
		}
	}

	// order filters
	sort.Slice(actx.befores, func(i, j int) bool {
		return actx.befores[i].name < actx.befores[j].name
	})

	sort.Slice(actx.afters, func(i, j int) bool {
		return actx.afters[i].name > actx.afters[j].name
	})

	return actx
}

func genType(m reflect.Method) string {
	outs := m.Type.NumOut()
	if outs == 0 || outs > 2 {
		panic("Gen functions must return 1 or 2 values")
	}
	return m.Type.Out(0).String()
}

var errStop = errors.New("stop")
