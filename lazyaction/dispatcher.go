package lazyaction

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	"github.com/felixge/httpsnoop"
	"golazy.dev/lazyaction/internal/args"
	"golazy.dev/lazyaction/router"
	"golazy.dev/lazyassets"
)

type Dispatcher struct {
	router *router.Router[Action]
	Files  *lazyassets.Manager
}

func (r *Dispatcher) String() string {
	//return r.router.String()
	return ""
}

/*
Route adds routes to the router

	Router.Route("/", func()string{return "Hello"}) // If verb is omited it is assumed is a GET
	Router.Route("POST", "/users", func() string{ return "User created!"})  // Verb can be added as a string
*/
func (d *Dispatcher) Route(def string, target any) {
	if d.router == nil {
		d.router = router.NewRouter[Action]()
	}

	path := def
	method := "GET"

	if strings.Contains(def, " ") {
		parts := strings.Split(def, " ")
		method = parts[0]
		path = parts[1]
	}

	u, err := url.Parse(path)
	if err != nil {
		panic(err)
	}

	action := &Action{
		Method:     method,
		URL:        *u,
		Name:       "Annonymous",
		Generators: &map[string][]args.Gen{},
	}

	switch t := target.(type) {
	case http.HandlerFunc:
		action.Handler = t
	case http.Handler:
		action.Handler = t.ServeHTTP
	default:
		if reflect.ValueOf(t).Kind() == reflect.Func {
			action.Fn = args.NewFn(t)
			break
		}
		panic(fmt.Sprintf("invalid argument type: %T", t))
	}

	d.router.Add(def, action)
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
func (d *Dispatcher) Resource(target any, options ...*ResourceOptions) {
	if d.router == nil {
		d.router = router.NewRouter[Action]()
	}
	if len(options) == 0 {
		options = append(options, &ResourceOptions{})
	}
	resource, err := newResource(target, options[0])
	if err != nil {
		panic(err)
	}
	for _, action := range resource.Actions() {
		d.router.Add(action.Method+" "+action.URL.String(), action)
	}

}

func (d *Dispatcher) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if d.router == nil {
		http.NotFound(w, req)
		return
	}
	action := d.router.Find(req)
	if action == nil {
		http.NotFound(w, req)
		return
	}

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		d.dispatch(action, w, req)
	})

	metrics := httpsnoop.CaptureMetrics(h, w, req)
	fmt.Printf("action : %+v\n", action)
	fmt.Printf("metrics: %+v\n", metrics)

}
