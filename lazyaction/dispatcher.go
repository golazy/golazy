package lazyaction

import (
	"net/http"

	"golazy.dev/lazyaction/router"
	"golazy.dev/lazyassets"
)

type Dispatcher struct {
	router *router.Router[Action]
	Assets *lazyassets.Assets
}

func (r *Dispatcher) String() string {
	//return r.router.String()
	return ""
}

type Route struct {
	Method string `json:"method"`
	URL    string `json:"url"`
	Name   string `json:"name"`
}

func (r *Dispatcher) Routes() []Route {
	all := r.router.All()
	routes := make([]Route, len(all))

	for i, route := range r.router.All() {
		routes[i].Method = route.Req.Method
		routes[i].URL = route.Req.URL.String()
		routes[i].Name = route.T.Name
	}

	return routes
}

/*
Route adds routes to the router

	Router.Route("/", func()string{return "Hello"}) // If verb is omited it is assumed is a GET
	Router.Route("POST", "/users", func() string{ return "User created!"})  // Verb can be added as a string
*/
func (d *Dispatcher) Route(def string, target any) {
	rc := &Constraints{d: d}
	rc.Route(def, target)
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
func (d *Dispatcher) Resource(target any, options ...*ResourceOptions) *Constraints {
	rc := &Constraints{d: d}

	return rc.Resource(target, options...)
}

func (d *Dispatcher) With(c Constraints) *Constraints {
	c2 := c
	c2.d = d
	return &c2
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
	action.ServeHTTP(w, req)

	//metrics := httpsnoop.CaptureMetrics(action, w, req)
	//fmt.Printf("action : %+v\n", action)
	//fmt.Printf("metrics: %+v\n", metrics)

}
