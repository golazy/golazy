package lazydispatch

import (
	"net/http"
)

type toSuper struct {
	scope *Scope
	as    string
	h     http.Handler
}

func (t *toSuper) As(route_name string) *toSuper {
	t.as = route_name
	return t
}

func (s *Scope) To(h http.Handler) {
	t := &toSuper{
		scope: s,
		h:     h,
	}
	s.addrgen(t)
}

func (t *toSuper) routes() []*Route {
	r := &Route{
		Handler: t.h,
	}

	s := t.scope.newChild()
	s.as = t.as
	r.Method, r.URL, r.Name, _, r.Models = s.routeInfo()

	r.normalize()

	return []*Route{r}
}
