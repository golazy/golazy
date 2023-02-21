package router

import (
	"net/http"
	"strings"

	"golazy.dev/lazysupport"
)

type Router[T any] struct {
	Routes       []*Route
	treeByMethod map[int]*routeTree[Route]
}

func (r *Router[T]) String() string {
	t := lazysupport.Table{
		Header: []string{"Name", "Verb", "Path"},
		Values: [][]string{},
	}
	for _, route := range r.Routes {
		t.Values = append(t.Values, []string{route.Name, route.Verb, route.Path})
	}

	return t.String()
}

func NewRouter[T any]() *Router[T] {

	r := &Router[T]{
		Routes:       []*Route{},
		treeByMethod: make(map[int]*routeTree[Route], len(Methods)),
	}
	for i := range Methods {
		r.treeByMethod[i] = &routeTree[Route]{}
	}
	return r
}

func (r *Router[T]) Add(route *Route) {
	r.Routes = append(r.Routes, route)

	for _, verb := range strings.Split(route.Verb, "|") {
		i := IsMethod(verb)
		if i < 0 {
			panic("Invalid verb: " + verb)
		}
		rt := r.treeByMethod[i]
		rt.Add(route.Path, route)
	}
}

func (r *Router[T]) Find(req *http.Request) *Route {
	i := IsMethod(req.Method)
	if i < 0 {
		panic("Invalid verb: " + req.Method)
	}
	rt := r.treeByMethod[i]
	return rt.Find(req.URL.Path)
}

/*

// Route routes a path to an action inside a controller
// Route("GET", "/posts", "posts#index")
// Route(posts.Controller)
// Route("/posts/:post_id", comments.Controller) // Nested Resource
func (r *Router[T]) Route(args ...any) {
	var verb string
	var controller any
	var action any
	var path string
	var target any
	// Add the option of having a standard HandlerFunc or Handler

	for _, arg := range args {
		k := reflect.ValueOf(arg).Kind()
		if k == reflect.Ptr {
			k = reflect.ValueOf(arg).Elem().Kind()
		}
		switch k {
		case reflect.String:
			for _, m := range Methods {
				if strings.ToUpper(arg.(string)) == m {
					verb = m
					continue
				}
				path = arg.(string)
			}
		case reflect.Struct:
			controller = arg
		case reflect.Func:
			action = arg
		default:
			panic(fmt.Sprintf("Invalid argument type: %s", k))
		}
	}

	if controller == nil && path == "" {
		panic("A path or a controller is required")
	}
	if controller != nil && action != nil {
		panic("Route requires either a controller or an action")
	}


	r.Add(&Route[T]{
		Verb:        verb,
		Path:        path,
		Controller:  controller,
		Action:      action,
		RouteName:   getFunctionName(action),
		ParamsNames: lazysupport.GetParamsNames(action),
		Target: 	controller,
	})


}

func (r *Router[T]) AddRoute(route *Route[T]) {
	r.Routes = append(r.Routes, route)

	for _, verb := range strings.Split(route.Verb, "|") {
		i := methodIndex(verb)
		if i < 0 {
			panic("Invalid verb: " + verb)
		}
		rt := r.treeByMethod[i]
		rt.Add(route.Path, route)
	}
}

func getFunctionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

/*
func (r *Router[T]) AddResource(resource *Resource) {
	r.AddResource(NewResource(resource))
}

func (r *Router[T]) AddResource(resource *Resource) {
	for _, action := range resource.Actions {
		r.Add(action.Verb, action.Path, action.RouteName, action.Destination, action)
	}
}
*/

/*

func (r *Router[T]) Find(req *http.Request) *T {
	l := len(req.URL.Path)
	if l < 1 || req.URL.Path[0] != '/' {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(http.StatusText(http.StatusBadRequest)))
		return
	}
	if l > 1 && req.URL.Path[l-1] == '/' {
		http.Redirect(w, req, req.URL.Path[0:l-1], http.StatusPermanentRedirect)
		return
	}
	i := methodIndex(req.Method)
	if i < 0 {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte(http.StatusText(http.StatusMethodNotAllowed)))
		return
	}

	rt := r.treeByMethod[i]
	t := rt.Find(req.URL.Path)
	if t == nil {
		return nil
	}
	return t.Target
}

// LinkTo(user1, "posts", "new") => "/users/1/posts/new"
// LinkTo(post1, comment1, "edit") => "/posts/1/comments/1/edit"
// LinkTo("posts", "new") => "/posts/new"
// LinkTo("posts", "publish") => "/posts/publish"
func (r *Router[T]) LinkTo(name string, params ...any) string {
	return ""
}

*/
