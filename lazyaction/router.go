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
	var target any
	var ins []string
	var outs []string

	for _, arg := range arguments {
		switch tArg := arg.(type) {
		case http.Handler:
			target = tArg.ServeHTTP
		case http.HandlerFunc:
			target = tArg
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
		}
	}
	if verb == "" {
		verb = "GET"
	}

	route := &router.Route{
		Verb:   verb,
		Path:   path,
		Target: target,
		Args:   ins,
		Rets:   outs,
		Name:   "Annonymous",
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
	a.Dispatch(route, w, r)
}

func (a *Routes) Dispatch(route *router.Route, w http.ResponseWriter, r *http.Request) {
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
