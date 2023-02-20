package lazyaction

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"golazy.dev/lazyaction/internal/router"
	"golazy.dev/lazydev"
)

type Router struct {
	router *router.Router[any]
}

func (r *Router) String() string {
	return r.router.String()
}

func (r *Router) Route(args ...any) {
	if r.router == nil {
		r.router = router.NewRouter[any]()
	}
	var verb string
	var path string
	var target any

	for i, arg := range args {
		k := reflect.ValueOf(arg).Kind()
		if k == reflect.Ptr {
			k = reflect.ValueOf(arg).Elem().Kind()
		}
		switch k {
		case reflect.String:
			for _, m := range router.Methods {
				if strings.ToUpper(arg.(string)) == m {
					verb = m
					continue
				}
				if strings.HasPrefix(arg.(string), "/") {
					path = arg.(string)
					continue
				}
				panic(fmt.Sprintf("Invalid path: %q", arg.(string)))
			}
		case reflect.Func:
			target = arg
		case reflect.Struct:
			// create a slice with all the elements of args minus the current element
			// and pass it to routeResource

			r.routeResource(arg, append(args[:i], args[i+1:]...))
			return

		default:
			panic(fmt.Sprintf("Invalid argument type: %s", k))
		}
	}
	if verb == "" {
		verb = "GET"
	}

	route := &router.Route[any]{
		Verb:   verb,
		Path:   path,
		Target: target,
	}

	r.router.Add(route)

}

func (r *Router) routeResource(resource any, options ...any) {

}

func (a *Router) ListenAndServe() error {
	server := &lazydev.Server{
		BootMode:  lazydev.ParentMode,
		HTTPAddr:  ":3000",
		HTTPSAddr: ":3000",
	}

	return server.ListenAndServe()
}

func (a *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if a.router == nil {
		panic("No routes defined")
	}
	route := a.router.Find(r)
	a.Dispatch(route, w, r)
}

func (a *Router) Dispatch(route *router.Route[any], w http.ResponseWriter, r *http.Request) {
	val := reflect.ValueOf(route.Target)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Func {
		panic("Invalid route target")
	}

	//prepare args
	mType := val.Type()
	ins := make([]reflect.Value, mType.NumIn())

	seenStrings := 0

	for i := 0; i < mType.NumIn(); i++ {
		inType := mType.In(i).String()
		switch inType {
		case "string":
			arg := ExtractParam2(r.URL.Path, 1)
			ins[i] = reflect.ValueOf(arg)
			seenStrings++
		case "lazyaction.ResponseWriter":
			ins[i] = reflect.ValueOf(ResponseWriter{w})
		case "*lazyaction.Request":
			ins[i] = reflect.ValueOf(&Request{r})
		case "lazyaction.Request":
			panic("Should use *http.Request")
		case "http.ResponseWriter":
			ins[i] = reflect.ValueOf(w)
		case "*http.Request":
			ins[i] = reflect.ValueOf(r)
		case "http.Request":
			panic("Should use *http.Request")
		default:
			panic(fmt.Sprintf("Can't fill the argument of type %s for %s", inType, ""))
		}
	}

	outs := val.Call(ins)
	for i := 0; i < mType.NumOut(); i++ {
		switch mType.Out(i).String() {
		case "error":
			if !outs[i].IsNil() {
				panic(val.Interface().(error))
			}
		case "string":
			w.Write([]byte(outs[i].String()))
		case "[]byte":
			w.Write(outs[i].Bytes())
		}
	}
}

func ExtractParam2(url string, paramPosition int) string {
	components := strings.Split(string(url)[1:], "/")
	for _, p := range components {
		if !strings.HasPrefix(p, ":") {
			continue
		}
		if paramPosition == 1 {
			return p
		}
		paramPosition--
	}
	return ""
}
