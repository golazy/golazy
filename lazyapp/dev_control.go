//go:build !lazydev

package lazyapp

import (
	"context"

	"golazy.dev/lazycontroller"
	"golazy.dev/lazycontrolplane"
)

func lazyDevContext(ctx context.Context) context.Context {
	return ctx
}

func lazyDevControlPlane(controlPlane *lazycontrolplane.ControlPlane, _ *lazycontroller.Renderer) *lazycontrolplane.ControlPlane {
	return controlPlane
}
