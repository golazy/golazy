package lazydispatch

import (
	"fmt"
	"net/http"
)

type Middleware interface {
	Handler(next http.Handler) http.Handler
}

type MiddlewareFunc func(http.Handler) http.Handler

func (fn MiddlewareFunc) Handler(next http.Handler) http.Handler {
	return fn(next)
}

type Dispatcher struct {
	middlewares []Middleware
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{}
}

func (d *Dispatcher) Use(middleware Middleware) {
	if middleware == nil {
		panic(fmt.Errorf("lazydispatch: middleware is nil"))
	}
	d.middlewares = append(d.middlewares, middleware)
}

func (d *Dispatcher) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	d.Handler(http.NotFoundHandler()).ServeHTTP(w, r)
}

func (d *Dispatcher) Handler(next http.Handler) http.Handler {
	if next == nil {
		next = http.NotFoundHandler()
	}
	for i := len(d.middlewares) - 1; i >= 0; i-- {
		next = d.middlewares[i].Handler(next)
	}
	return next
}
