/*
package lazyaction implements a http router that support url params and http verbs

It was design to use together lazyaction/controller:

But it can be used alone:
*/
package lazyaction

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/golazy/golazy/lazysupport"
)

var methods = []string{"GET", "POST", "OPTIONS", "PUT", "PATCH", "DELETE", "WS"}

func methodIndex(method string) int {
	for i, m := range methods {
		if m == method {
			return i
		}
	}
	return -1

}

type RouteAction interface {
	http.Handler
	fmt.Stringer
}

type Router struct {
	byMethod map[int]*routingTable[RouteAction]
}

func NewRouter() *Router {

	r := &Router{
		byMethod: make(map[int]*routingTable[RouteAction], len(methods)),
	}
	for i := range methods {
		r.byMethod[i] = &routingTable[RouteAction]{}
	}
	return r
}

func (r *Router) Add(method, path string, handler RouteAction) {
	i := methodIndex(method)
	if i < 0 {
		panic("Method can only be " + lazysupport.ToSentence("or ", methods...) + ".Got " + method)
	}

	rt := r.byMethod[i]
	rt.Add(path, &handler)
}

type byPath [][]string

func (a byPath) Len() int      { return len(a) }
func (a byPath) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byPath) Less(i, j int) bool {

	k := strings.Compare(
		strings.ReplaceAll(a[i][1], ":", "|"),
		strings.ReplaceAll(a[j][1], ":", "|")) // For wildchards to be last
	if k != 0 {
		return k < 0
	}
	var iPos, jPos int
	for l, m := range methods {
		if m == a[i][0] {
			iPos = l
		}
		if m == a[j][0] {
			jPos = l
		}
	}
	return iPos < jPos

}

func (r *Router) Table() *lazysupport.Table {
	t := lazysupport.Table{
		Header: []string{"METHOD", "PATH", "DESTINATION"},
		Values: [][]string{},
	}

	for i, table := range r.byMethod {
		for _, route := range table.Routes() {
			t.Values = append(t.Values, []string{methods[i], route.path, (*route.t).String()})
		}
	}

	sort.Sort(byPath(t.Values))

	return &t
}

func (r *Router) String() string {
	return r.Table().String()
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

	rt := router.byMethod[i]
	h := rt.Find(r.URL.Path)
	if h != nil {
		(*h).ServeHTTP(w, r)
		return
	}

	http.NotFound(w, r)

}

func (r *Router) Resource(c interface{}) {
	controller := newController(c)
	for _, route := range controller.Routes() {
		for _, method := range strings.Split(route.Verb, "|") {
			path := "/" + route.Controller.name
			if route.Path != "" {
				path = path + "/" + route.Path
			}
			r.Add(method, path, route)
		}
	}
}
