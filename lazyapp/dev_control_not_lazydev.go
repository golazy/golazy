//go:build !lazydev

package lazyapp

import (
	"context"

	"golazy.dev/lazyassets"
	"golazy.dev/lazycache"
	"golazy.dev/lazycontroller"
	"golazy.dev/lazycontrolplane"
	"golazy.dev/lazydeps"
	"golazy.dev/lazyjobs"
	"golazy.dev/lazyroutes"
)

func lazyDevContext(ctx context.Context) context.Context {
	return ctx
}

func lazyDevControlPlane(controlPlane *lazycontrolplane.ControlPlane, _ *lazycontroller.Renderer, _ *lazyroutes.Scope, _ *lazyassets.Registry, _ *lazycache.Cache, _ *lazydeps.Scope, _ *lazyjobs.JobRunner) *lazycontrolplane.ControlPlane {
	return controlPlane
}

func (app *App) controlPlaneInServeHTTP() *lazycontrolplane.ControlPlane {
	if app.ControlPlane == nil {
		return nil
	}
	controlAddr, controlAddrSet := controlPlaneListenAddr()
	if !controlAddrSet || !sameListenAddr(listenAddr(), controlAddr) {
		return nil
	}
	return app.ControlPlane
}

func (app *App) controlPlaneWithoutListenAddress() *lazycontrolplane.ControlPlane {
	return nil
}
