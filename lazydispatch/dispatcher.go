package lazydispatch

import (
	"fmt"
	"log/slog"
	"net/http"
	"reflect"
	"strings"

	"golazy.dev/lazytelemetry/lazytracing"
)

type Middleware interface {
	Handler(next http.Handler) http.Handler
}

type MiddlewareFunc func(http.Handler) http.Handler

func (fn MiddlewareFunc) Handler(next http.Handler) http.Handler {
	return fn(next)
}

func (MiddlewareFunc) MiddlewareName() string {
	return "lazydispatch.MiddlewareFunc"
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
		next = instrumentMiddleware(d.middlewares[i], next, i)
	}
	return next
}

func instrumentMiddleware(middleware Middleware, next http.Handler, index int) http.Handler {
	handler := middleware.Handler(next)
	name := middlewareName(middleware)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := lazytracing.StartRegion(r.Context(), "middleware "+name,
			slog.String("middleware.name", name),
			slog.Int("middleware.index", index),
		)
		if span == nil {
			handler.ServeHTTP(w, r)
			return
		}
		defer span.End()
		handler.ServeHTTP(w, r.WithContext(ctx))
	})
}

func middlewareName(middleware Middleware) string {
	if middleware == nil {
		return "nil"
	}
	if named, ok := middleware.(interface{ MiddlewareName() string }); ok {
		if name := strings.TrimSpace(named.MiddlewareName()); name != "" {
			return name
		}
	}
	t := reflect.TypeOf(middleware)
	if t == nil {
		return "unknown"
	}
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.PkgPath() != "" && t.Name() != "" {
		return t.PkgPath() + "." + t.Name()
	}
	return strings.TrimPrefix(reflect.TypeOf(middleware).String(), "*")
}
