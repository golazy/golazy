package router

import (
	"net/http"
	"strings"
)

type methodMatcher[T any] struct {
	get     Matcher[T]
	post    Matcher[T]
	put     Matcher[T]
	delete  Matcher[T]
	patch   Matcher[T]
	options Matcher[T]
}

func newMethodMatcher[T any]() Matcher[T] {
	return &methodMatcher[T]{
		get:     newSchemeMatcher[T](),
		post:    newSchemeMatcher[T](),
		put:     newSchemeMatcher[T](),
		delete:  newSchemeMatcher[T](),
		patch:   newSchemeMatcher[T](),
		options: newSchemeMatcher[T](),
	}
}

func (vm *methodMatcher[T]) All() []Route[T] {
	all := []Route[T]{}

	fill := func(method string, m Matcher[T]) {
		for _, r := range m.All() {
			r.Req.Method = method
			all = append(all, r)
		}
	}

	fill(http.MethodGet, vm.get)
	fill(http.MethodPost, vm.post)
	fill(http.MethodPut, vm.put)
	fill(http.MethodDelete, vm.delete)
	fill(http.MethodPatch, vm.patch)
	fill(http.MethodOptions, vm.options)

	return all
}
func (vm *methodMatcher[T]) Add(req *RouteDefinition, t *T) {
	eachMethod(req.Method, func(method string) {
		switch method {
		case http.MethodGet:
			vm.get.Add(req, t)
		case http.MethodPost:
			vm.post.Add(req, t)
		case http.MethodPut:
			vm.put.Add(req, t)
		case http.MethodDelete:
			vm.delete.Add(req, t)
		case http.MethodPatch:
			vm.patch.Add(req, t)
		case http.MethodOptions:
			vm.options.Add(req, t)
		}
	})
}

func eachMethod(s string, fn func(string)) {
	if s == "*" {
		for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch, http.MethodOptions} {
			fn(method)
		}
		return
	}
	for _, method := range strings.Split(s, ",") {
		fn(method)
	}
}

func (vm *methodMatcher[T]) Find(req *http.Request) *T {
	switch req.Method {
	case http.MethodGet:
		return vm.get.Find(req)
	case http.MethodPost:
		return vm.post.Find(req)
	case http.MethodPut:
		return vm.put.Find(req)
	case http.MethodDelete:
		return vm.delete.Find(req)
	case http.MethodPatch:
		return vm.patch.Find(req)
	case http.MethodOptions:
		return vm.options.Find(req)
	case "":
		panic("Request without method!!!")
	default:
		panic("Unknown method " + req.Method)
	}
}
