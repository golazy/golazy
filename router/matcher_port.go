package router

import (
	"net/http"
	"strconv"
)

type portPaths[T any] struct {
	port  int
	paths Matcher[T]
}

// Port 0 matches all ports
type portMatcher[T any] struct {
	all   Matcher[T]
	ports []portPaths[T]
}

func newPortMatcher[T any]() Matcher[T] {
	return &portMatcher[T]{
		all:   NewPathMatcher[T](),
		ports: []portPaths[T]{},
	}
}

func (r *portMatcher[T]) All() []Route[T] {
	all := []Route[T]{}

	for _, portRoutes := range r.ports {
		for _, path := range portRoutes.paths.All() {
			path.Req.URL.Host = path.Req.URL.Hostname() + ":" + strconv.Itoa(portRoutes.port)
			all = append(all, path)
		}
	}
	for _, path := range r.all.All() {
		path.Req.URL.Host = path.Req.URL.Hostname()
		all = append(all, path)
	}
	return all
}
func (r *portMatcher[T]) Add(req *RouteDefinition, t *T) {
	port := req.Port
	portN, _ := strconv.Atoi(port)

	paths := r.all
	if portN == 0 {
		paths.Add(req, t)
		return
	}

	for _, p := range r.ports {
		if p.port == portN {
			(p.paths).Add(req, t)
			return
		}
	}

	p := portPaths[T]{
		port:  portN,
		paths: NewPathMatcher[T](),
	}
	paths = p.paths
	paths.Add(req, t)
	r.ports = append(r.ports, p)
}

func (r portMatcher[T]) Find(u *http.Request) *T {
	var target *T
	port, _ := strconv.Atoi(u.URL.Port())

	if port != 0 {
		for _, pp := range r.ports {
			if pp.port == port {
				target = pp.paths.Find(u)
				if target != nil {
					return target
				}
				break
			}
		}
	}

	return r.all.Find(u)
}
