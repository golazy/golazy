package lazydispatch

import (
	"fmt"
)

type routeGen interface {
	routes() []*Route
}

func newScope() *Scope {
	return &Scope{}
}

type Scope struct {
	parent    *Scope
	rgens     []routeGen
	as        string
	path      string
	method    string
	model     any
	verb      string
	namespace string
}

func (s *Scope) clone() *Scope {
	s2 := *s
	return &s2
}
func (s *Scope) newChild() *Scope {
	return &Scope{parent: s}
}
func (s *Scope) top() *Scope {
	for s.parent != nil {
		s = s.parent
	}
	return s
}

func (s *Scope) routes() []*Route {
	var routes []*Route

	for _, e := range s.top().rgens {
		routes = append(routes, e.routes()...)
	}
	return routes
}

// addrgen adds a route geneator top scope
func (s *Scope) addrgen(r routeGen) {
	s = s.top()
	s.rgens = append(s.rgens, r)
}

func (s *Scope) Path(p string) *Scope {
	s = s.newChild()
	s.path = p
	return s
}

func (s *Scope) As(a string) *Scope {
	s = s.newChild()
	s.as = a
	return s
}
func (s *Scope) newMethod(m string) *Scope {
	s = s.newChild()
	s.method = m
	return s
}

func (s *Scope) Get(path string) *Scope {
	return s.newMethod("GET").Path(path)
}
func (s *Scope) Post(path string) *Scope {
	return s.newMethod("POST").Path(path)
}
func (s *Scope) Put(path string) *Scope {
	return s.newMethod("PUT").Path(path)
}
func (s *Scope) Patch(path string) *Scope {
	return s.newMethod("PATCH").Path(path)
}
func (s *Scope) Delete(path string) *Scope {
	return s.newMethod("DELETE").Path(path)
}
func (s *Scope) Options(path string) *Scope {
	return s.newMethod("OPTIONS").Path(path)
}

func (s *Scope) Namespace(n string) *Scope {
	s = s.newChild()
	s.namespace = n
	return s
}

func (s *Scope) Draw(fn func(s *Scope)) *Scope {
	s = s.newChild()
	fn(s)
	return s
}

func (s *Scope) routeInfo() (method, path, name, namespace string, models []any) {
	method, path, name, namespace = s.method, s.path, s.as, s.namespace

	if s.model != nil {
		models = []any{s.model}
	}

	for s := s.parent; s != nil; s = s.parent {
		path = joinWithChar("/", s.path, path)
		name = joinWithChar("_", s.as, name)
		namespace = joinWithChar("/", s.namespace, namespace)
		if len(models) > 0 && s.model != nil {
			models = append([]any{s.model}, models...)
		}
		if method == "" {
			method = s.method
		} else {
			if s.method != "" && s.method != method {
				panic(fmt.Sprintf("can't add %s %s as %s in %q namespace with a constraint of %s method", method, path, name, namespace, s.method))
			}
		}
	}
	if method == "" {
		method = "GET"
	}

	if path == "" {
		path = "/"
	}
	if path[0] != '/' {
		path = "/" + path
	}
	name = joinWithChar("_", s.verb, name)
	return
}
