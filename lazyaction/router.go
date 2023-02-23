package lazyaction

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"golazy.dev/lazyaction/internal/args"
	"golazy.dev/lazyaction/internal/router"
)

type Routes struct {
	router *router.Router[Action]
}

func (r *Routes) String() string {
	//return r.router.String()
	return ""
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
		r.router = router.NewRouter[Action]()
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

	action := &Action{
		Verb:       verb,
		Path:       path,
		Handler:    handler,
		Fn:         target,
		Name:       "Annonymous",
		Generators: &map[string][]args.Gen{},
	}

	r.router.Add(verb, path, action)
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
		r.router = router.NewRouter[Action]()
	}
	if len(options) == 0 {
		options = append(options, &ResourceOptions{})
	}
	resource, err := newResource(target, options[0])
	if err != nil {
		panic(err)
	}
	for _, action := range resource.Actions() {
		r.router.Add(action.Verb, action.Path, action)
	}

}

func (r *Routes) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if r.router == nil {
		panic("No routes defined")
	}
	action := r.router.Find(req.Method, req.URL.Path)
	if action == nil {
		panic("No route found")
	}

	r.dispatch(action, w, req)
}
