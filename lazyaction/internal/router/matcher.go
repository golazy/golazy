package router

import (
	"fmt"
	"net/http"
)

type Route[T any] struct {
	Req *http.Request
	T   *T
}

func (r Route[T]) String() string {
	return fmt.Sprintf("%v %v => %v", r.Req.Method, r.Req.URL.String(), *r.T)
}

type Matcher[T any] interface {
	Add(r *RouteDefinition, t *T)
	Find(*http.Request) *T
	All() []Route[T]
}
