package lazydispatch

import (
	"net/http"
	"net/url"
	"sync"

	"golazy.dev/router"
)

type Dispatcher struct {
	Routes      []*Route
	httpr       *router.Router[Route]
	names       *namedRoutes
	middlewares []func(http.Handler) http.Handler
	app         func() http.Handler
}

func New() *Dispatcher {
	d := &Dispatcher{
		httpr:       router.NewRouter[Route](),
		names:       newNamedRoutes(),
		Routes:      make([]*Route, 0),
		middlewares: make([]func(http.Handler) http.Handler, 0),
	}
	d.app = sync.OnceValue(func() http.Handler {

		var handler http.Handler = http.HandlerFunc(d.dispatch)
		for _, m := range d.middlewares {
			handler = m(handler)
		}
		return handler
	})
	return d
}

func (d *Dispatcher) dispatch(w http.ResponseWriter, r *http.Request) {
	route := d.httpr.Find(r)
	if route == nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(http.StatusText(http.StatusNotFound)))
		return
	}
	route.Handler.ServeHTTP(w, r)
}

// Use adds a middleware to the dispatcher
func (d *Dispatcher) Use(middleware func(http.Handler) http.Handler) {
	// All the middlewares have to be setup before calling ServeHTTP
	d.middlewares = append(d.middlewares, middleware)
}

func (d *Dispatcher) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	d.app().ServeHTTP(w, r)
}

func (d *Dispatcher) PathFor(args ...any) string {
	return d.names.PathFor(args...)
}

func (d *Dispatcher) Draw(fn func(r *Scope)) *Scope {
	drawer := newScope()
	fn(drawer)
	d.Routes = drawer.routes()
	for _, route := range d.Routes {

		route.normalize()
		// Add route
		if route.Handler != nil {
			d.httpr.Add(&router.RouteDefinition{
				Method: route.Method,
				Path:   route.URL,
			}, route)
		}

		// Add name
		if route.Name != "" {
			u, err := url.Parse(route.URL)
			if err != nil {
				panic(err)
			}

			d.names.Add(Route{
				Path:   u.Path,
				Name:   route.Name,
				Models: route.Models,
			})
		}
	}

	// Add to names
	return drawer

}
