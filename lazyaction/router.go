package lazyaction

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"golazy.dev/lazyaction/internal/router"
	"golazy.dev/lazydev"
)

type Routes struct {
	router *router.Router[any]
}

func (r *Routes) String() string {
	return r.router.String()
}

type Router interface {
	Route(args ...any)
	Resource(controller any, opts ...ResourceOptions)
}

/*
Route adds routes to the router

	Router.Route("/", func()string{return "Hello"}) // If verb is omited it is assumed is a GET
	Router.Route("POST", "/users", func() string{ return "User created!"})  // Verb can be added as a string
*/
func (r *Routes) Route(arguments ...any) {
	if r.router == nil {
		r.router = router.NewRouter[any]()
	}
	var verb string
	var path string
	var target reflect.Value
	var handler http.HandlerFunc
	var ins []string
	var outs []string

	for _, arg := range arguments {
		switch tArg := arg.(type) {
		case http.HandlerFunc:
			handler = tArg
		case http.Handler:
			handler = tArg.ServeHTTP
		case string:
			if router.IsMethod(tArg) != -1 {
				verb = tArg
				continue
			}
			if strings.HasPrefix(tArg, "/") {
				path = tArg
				continue
			}
			panic(fmt.Sprintf("invalid path: %q", arg.(string)))
		default:
			if reflect.ValueOf(arg).Kind() == reflect.Func {
				target = reflect.ValueOf(arg)
				continue
			}
			panic(fmt.Sprintf("invalid argument type: %T", arg))
		}
	}
	if verb == "" {
		verb = "GET"
	}

	route := &router.Route{
		Verb:    verb,
		Path:    path,
		Target:  target,
		Handler: handler,
		Args:    ins,
		Rets:    outs,
		Name:    "Annonymous",
	}

	r.router.Add(route)
}

/*
	Resource adds a resource to the router

The resource name is extracted from the struct name minus the Controller suffix.
If the struct is called Controller it will use the package name.

Given a struct called UserController, the follwing methods will generate the following routes:

- Index()   => GET    /users
- New()     => GET    /users/new
- Create()  => POST   /users
- Show()    => GET    /users/:id
- Edit()    => GET    /users/:id/edit
- Update()  => PUT    /users/:id
- Destroy() => DELETE /users/:id

Custom actions can be added to the resource by combining the verb and if it belongs to a Member.

- Popular() => GET /users/popular
- MemberComments() => GET /users/:id/comments
- PutMemberSuspend() => PUT /users/:id/suspend

Resource internally calls Route to add the routes to the router.
*/
func (r *Routes) Resource(target any, options ...*ResourceOptions) {
	if r.router == nil {
		r.router = router.NewRouter[any]()
	}
	if len(options) == 0 {
		options = append(options, &ResourceOptions{})
	}
	resource, err := newResource(target, options[0])
	if err != nil {
		panic(err)
	}
	for _, route := range resource.Routes() {
		r.router.Add(route)
	}

}

func (a *Routes) ListenAndServe() error {
	server := &lazydev.Server{
		BootMode:  lazydev.ParentMode,
		HTTPAddr:  ":3000",
		HTTPSAddr: ":3000",
	}

	return server.ListenAndServe()
}

func (a Routes) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if a.router == nil {
		panic("No routes defined")
	}
	route := a.router.Find(r)
	if route == nil {
		panic("No route found")
	}
	if route.Handler != nil {
		route.Handler(w, r)
		return
	}

	ins := a.getInputs(route, w, r)
	if route.Target.IsZero() {
		panic("is zero")
	}
	if route.Target.IsNil() {
		panic("is nil")
	}
	if route.Target.IsValid() == false {
		panic("is invalid")
	}
	outs := route.Target.Call(ins)
	a.processOutputs(route, w, r, outs)
}

func (a *Routes) processOutputs(route *router.Route, w http.ResponseWriter, r *http.Request, outs []reflect.Value) {
	for _, out := range outs {
		switch out.Kind() {
		case reflect.String:
			w.Write([]byte(out.String()))
		case reflect.Int:
			w.WriteHeader(int(out.Int()))
		case reflect.Interface:
			if out.IsNil() {
				continue
			}
			switch tOut := out.Interface().(type) {
			case error:
			default:
				panic(fmt.Sprintf("invalid output type: %T", tOut))
			}
		default:
			panic(fmt.Sprintf("invalid output type: %T", out.Interface()))
		}
	}
}

func (a *Routes) getInputs(route *router.Route, w http.ResponseWriter, r *http.Request) []reflect.Value {
	ins := make([]reflect.Value, len(route.Args))
	for i, arg := range route.Args {
		switch arg {
		case "http.ResponseWriter":
			ins[i] = reflect.ValueOf(w)
		case "*http.Request":
			ins[i] = reflect.ValueOf(r)
		default:
			ins[i] = reflect.ValueOf(ExtractParam2(r.URL.Path, i+1))
		}
	}
	return ins
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
