package lazyaction

import (
	"fmt"
	"net/http"
	"strings"
)

type RouteDefinition struct {
	Verb           string
	Path           string
	Name           string
	Destination    string
	Handler        http.Handler
	ResourceName   string // comment or comments or post_comments
	ResourceMember bool   // true if it adds a member path
	ResourceAction string // "new" or "edit" or custom
	ParamsPosition []int
	Member         bool
}

func (rd *RouteDefinition) String() string {
	return fmt.Sprintf("%s %s %s %s", rd.Name, rd.Verb, rd.Path, rd.Destination)
}

type Router struct {
	Routes       []*RouteDefinition
	treeByMethod map[int]*routeTree[http.Handler]
}

var methods = []string{"GET", "POST", "OPTIONS", "PUT", "PATCH", "DELETE", "WS"}

func methodIndex(method string) int {
	for i, m := range methods {
		if m == method {
			return i
		}
	}
	return -1

}

func NewRouter() *Router {
	r := &Router{
		Routes:       []*RouteDefinition{},
		treeByMethod: make(map[int]*routeTree[http.Handler], len(methods)),
	}
	for i := range methods {
		r.treeByMethod[i] = &routeTree[http.Handler]{}
	}
	return r
}

func (r *Router) Add(verb, path, name, destination string, h http.Handler) {
	route := &RouteDefinition{
		Verb:        verb,
		Path:        path,
		Name:        name,
		Destination: destination,
		Handler:     h,
	}
	r.Routes = append(r.Routes, route)

	for _, verb := range strings.Split(route.Verb, "|") {
		i := methodIndex(verb)
		if i < 0 {
			panic("Invalid verb: " + verb)
		}
		rt := r.treeByMethod[i]
		rt.Add(path, &route.Handler)
	}
}

func (router *Router) AddResourceDefinition(r *ResourceDefinition) {
	router.AddResource(NewResource(r))
}

func (router *Router) AddResource(resource *Resource) {
	for _, action := range resource.Actions {
		router.Add(action.Verb, action.Path, action.RouteName, action.Destination, action)
	}
}

func (router *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	l := len(r.URL.Path)
	if l < 1 || r.URL.Path[0] != '/' {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(http.StatusText(http.StatusBadRequest)))
		return
	}
	if l > 1 && r.URL.Path[l-1] == '/' {
		http.Redirect(w, r, r.URL.Path[0:l-1], http.StatusPermanentRedirect)
		return
	}
	i := methodIndex(r.Method)
	if i < 0 {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte(http.StatusText(http.StatusMethodNotAllowed)))
		return
	}

	rt := router.treeByMethod[i]
	h := rt.Find(r.URL.Path)
	if h != nil {
		(*h).ServeHTTP(w, r)
		return
	}

	http.NotFound(w, r)
}

// LinkTo(user1, "posts", "new") => "/users/1/posts/new"
// LinkTo(post1, comment1, "edit") => "/posts/1/comments/1/edit"
// LinkTo("posts", "new") => "/posts/new"
// LinkTo("posts", "publish") => "/posts/publish"
func (r *Router) LinkTo(name string, params ...any) string {

	return ""
}
