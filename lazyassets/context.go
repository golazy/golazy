package lazyassets

import "context"

type registryContextKey struct{}

// WithRegistry returns a context carrying registry.
func WithRegistry(ctx context.Context, registry *Registry) context.Context {
	return context.WithValue(ctx, registryContextKey{}, registry)
}

// FromContext returns the asset registry carried by ctx.
func FromContext(ctx context.Context) (*Registry, bool) {
	if ctx == nil {
		return nil, false
	}
	registry, ok := ctx.Value(registryContextKey{}).(*Registry)
	return registry, ok && registry != nil
}
