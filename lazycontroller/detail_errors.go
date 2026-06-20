package lazycontroller

import "context"

type detailErrorsContextKey struct{}

func WithDetailErrors(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, detailErrorsContextKey{}, true)
}

func DetailErrors(ctx context.Context) bool {
	if ctx != nil {
		if force, ok := ctx.Value(detailErrorsContextKey{}).(bool); ok && force {
			return true
		}
	}
	return defaultDetailErrors()
}
