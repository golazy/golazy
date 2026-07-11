package lazycontroller

import (
	"context"
	"fmt"

	"golazy.dev/lazypath"
)

// PathForFunc builds a path from a named route and route parameter values.
type PathForFunc func(name string, values ...any) (string, error)

// URLParams appends query parameters to a generated route path.
type URLParams = lazypath.URLParams

type pathForContextKey struct{}

// WithPathFor returns a context carrying the application's route path helper.
func WithPathFor(ctx context.Context, pathFor PathForFunc) context.Context {
	return context.WithValue(ctx, pathForContextKey{}, pathFor)
}

// PathFor builds a path from a named route and route parameter values.
func (b *Base) PathFor(name string, values ...any) (string, error) {
	pathFor, ok := pathForFromContext(b.ctx)
	if !ok {
		pathFor, ok = pathForFromContext(b.appCtx)
	}
	if !ok {
		return "", fmt.Errorf("controller path helper is missing")
	}
	routeValues, params := lazypath.SplitValues(values)
	path, err := pathFor(name, routeValues...)
	if err != nil {
		return "", err
	}
	return lazypath.AppendURLParams(path, params), nil
}

// MustPathFor builds a path and panics when the route cannot be generated.
func (b *Base) MustPathFor(name string, values ...any) string {
	path, err := b.PathFor(name, values...)
	if err != nil {
		panic(err)
	}
	return path
}

// Param returns the named route parameter from the current request.
func (b *Base) Param(name string) string {
	if b == nil || b.request == nil {
		return ""
	}
	return b.request.PathValue(name)
}

func pathForFromContext(ctx context.Context) (PathForFunc, bool) {
	if ctx == nil {
		return nil, false
	}
	pathFor, ok := ctx.Value(pathForContextKey{}).(PathForFunc)
	if !ok || pathFor == nil {
		return nil, false
	}
	return pathFor, true
}
