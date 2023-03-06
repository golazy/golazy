package router

import "net/http"

func NewRouter[T any]() *Router[T] {
	mm := NewMethodMatcher[T]().(*MethodMatcher[T])
	r := Router[T](*mm)
	return &r
}

type Router[T any] MethodMatcher[T]

func (r Router[T]) All() []Route[T] {
	mm := MethodMatcher[T](r)
	return mm.All()
}

func (r Router[T]) Add(path string, t *T) {
	rd := NewRouteDefinition(path)
	mm := MethodMatcher[T](r)
	mm.Add(rd, t)
}

func (r Router[T]) Find(req *http.Request) *T {
	mm := MethodMatcher[T](r)
	return mm.Find(req)
}
