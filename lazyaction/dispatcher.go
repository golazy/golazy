package lazyaction

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	"golazy.dev/lazyaction/internal/args"
	"golazy.dev/lazyaction/internal/router"
	"golazy.dev/lazyview/static_files"
)

type Dispatcher struct {
	router router.Matcher[Action]
	Files  *static_files.Manager
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
func (d *Dispatcher) Route(arguments ...any) {
	if d.router == nil {
		d.router = router.NewRouter[Action]()
	}
	var verb string
	var path string
	var target *args.Fn
	var handler http.HandlerFunc

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
				target = args.NewFn(arg)
				continue
			}
			panic(fmt.Sprintf("invalid argument type: %T", arg))
		}
	}
	if verb == "" {
		verb = "GET"
	}

	u, err := url.Parse(path)
	if err != nil {
		panic(err)
	}

	action := &Action{
		Verb:       verb,
		URL:        *u,
		Handler:    handler,
		Fn:         target,
		Name:       "Annonymous",
		Generators: &map[string][]args.Gen{},
	}

	req, err := http.NewRequest(verb, path, nil)
	if err != nil {
		panic(err)
	}

	d.router.Add(req, action)
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

		req, err := http.NewRequest(action.Verb, action.URL.String(), nil)
		if err != nil {
			panic(err)
		}
		d.router.Add(req, action)
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

	d.dispatch(action, w, req)
}
