// Package router implements an http router
package router

import "net/http"

func NewRouter[T any]() *Router[T] {
	mm := newMethodMatcher[T]().(*methodMatcher[T])
	r := Router[T](*mm)
	return &r
}

type Router[T any] methodMatcher[T]

func (r Router[T]) All() []Route[T] {
	mm := methodMatcher[T](r)
	return mm.All()
}

func (r Router[T]) Add(req *RouteDefinition, t *T) {
	mm := methodMatcher[T](r)
	mm.Add(req, t)
}

func (r Router[T]) AddByPath(path string, t *T) {
	rd := NewRouteDefinition(path)
	r.Add(rd, t)
}

func (r Router[T]) Find(req *http.Request) *T {
	mm := methodMatcher[T](r)
	return mm.Find(req)
}
