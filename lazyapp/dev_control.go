//go:build !lazydev

package lazyapp

import (
	"golazy.dev/lazycontroller"
	"golazy.dev/lazycontrolplane"
)

func lazyDevControlPlane(controlPlane *lazycontrolplane.ControlPlane, _ *lazycontroller.Renderer) *lazycontrolplane.ControlPlane {
	return controlPlane
}
