//go:build !lazydev

package lazytelemetry

import "golazy.dev/lazytelemetry/lazytracing"

type requestCapture struct{}

func beginRequestCapture(bool, string) *requestCapture {
	return nil
}

func (capture *requestCapture) Finish(requestCaptureResult, *lazytracing.Span) {}
