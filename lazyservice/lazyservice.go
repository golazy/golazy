// Package lazyservice provides a framework for building lazy applications in Go.
// A lazy application is an application that starts and stops services on demand.
// It allows you to define services as functions and run them within the application.
// The lazyapp package provides an interface for defining services, adding values and types to the application, and running the application and its services.
// It also provides a default logger implementation and supports colored debug messages and JSON logs.
// The application uses trace regions for the app and each of the services.
package lazyservice

import (
	"context"
	"log/slog"

	"github.com/golazy/golazy/lazycontext"
)

// ServiceFunc creates a new service from a function.
func ServiceFunc(name string, f func(context.Context, *slog.Logger) error) Service {
	return &srvFn{
		name: name,
		f:    f,
	}
}

type srvFn struct {
	name string
	f    func(context.Context, *slog.Logger) error
}

func (f *srvFn) Run(ctx context.Context) error {
	l := lazycontext.Get[*slog.Logger](ctx)
	if l == nil {
		l = slog.Default()
	}
	lctx := lazycontext.Get[lazycontext.LazyContext](ctx)
	if lctx == nil {
		panic("adsf")
	}
	lazycontext.Set(ctx, l)

	return f.f(ctx, l)
}

func (f *srvFn) Desc() ServiceDescription {
	return serviceFuncDesc{name: f.name}
}

type serviceFuncDesc struct {
	name string
}

func (d serviceFuncDesc) Name() string {
	return d.name
}

type ServiceDescription interface {
	Name() string
}

type Service interface {
	Desc() ServiceDescription
	Run(context.Context) error
}
