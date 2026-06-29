//go:build !lazydev

package lazytracing

import "context"

// WithAllocationSampling is a no-op outside lazydev builds.
func WithAllocationSampling(ctx context.Context) context.Context {
	return ctx
}

func startSpanAllocationSample(context.Context, *Span) {}

func finishSpanAllocationSample(*Span) {}
