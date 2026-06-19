package lazycontroller

import (
	"context"
	"fmt"
)

// PathForFunc builds a path from a named route and route parameter values.
type PathForFunc func(name string, values ...any) (string, error)

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
	return pathFor(name, values...)
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
