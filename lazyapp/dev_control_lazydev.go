//go:build lazydev

package lazyapp

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"golazy.dev/lazyassets"
	"golazy.dev/lazybuildinfo"
	"golazy.dev/lazycache"
	"golazy.dev/lazycontroller"
	"golazy.dev/lazycontrolplane"
	"golazy.dev/lazydeps"
	"golazy.dev/lazyjobs"
	"golazy.dev/lazypwa"
	"golazy.dev/lazyroutes"
	"golazy.dev/lazytelemetry"
	"golazy.dev/lazyworkers"
)

const lazyDevReloadViewsPath = "/_golazy/views/reload"
const lazyDevControlViewsPath = "/views"

func lazyDevContext(ctx context.Context) context.Context {
	return lazycontroller.LazyDevContext(ctx)
}

func lazyDevControlPlane(controlPlane *lazycontrolplane.ControlPlane, renderer *lazycontroller.Renderer, router *lazyroutes.Scope, assets *lazyassets.Registry, cache *lazycache.Cache, dependencies *lazydeps.Scope, jobs *lazyjobs.JobRunner, workers *lazyworkers.Registry, pwa *lazypwa.App, runtime *runtimeState) *lazycontrolplane.ControlPlane {
	if controlPlane == nil {
		controlPlane = lazycontrolplane.New(lazycontrolplane.Config{})
	}
	if runtime != nil {
		controlPlane.AddReadinessCheck(lazycontrolplane.ReadinessCheck{
			Name:  "shutdown",
			Check: runtime.ReadinessCheck,
		})
	}
	registerLazyDevViewHandlers(controlPlane, renderer)
	lazyroutes.RegisterLazyDevHandlers(controlPlane, router)
	lazycontroller.RegisterLazyDevHandlers(controlPlane)
	lazybuildinfo.RegisterLazyDevHandlers(controlPlane)
	lazyassets.RegisterLazyDevHandlers(controlPlane, assets)
	lazycache.RegisterLazyDevHandlers(controlPlane, cache)
	lazydeps.RegisterLazyDevHandlers(controlPlane, dependencies, lazydeps.WithLazyDevRuntime(runtime))
	lazyjobs.RegisterLazyDevHandlers(controlPlane, jobs)
	lazyworkers.RegisterLazyDevHandlers(controlPlane, workers)
	lazypwa.RegisterLazyDevHandlers(controlPlane, pwa)
	lazytelemetry.RegisterLazyDevHandlers(controlPlane)
	return controlPlane
}

func (app *App) controlPlaneInServeHTTP() *lazycontrolplane.ControlPlane {
	return app.ControlPlane
}

func (app *App) controlPlaneWithoutListenAddress() *lazycontrolplane.ControlPlane {
	return app.ControlPlane
}

func registerLazyDevViewHandlers(controlPlane *lazycontrolplane.ControlPlane, renderer *lazycontroller.Renderer) {
	var reloadMu sync.Mutex
	reloadViews := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reloadMu.Lock()
		defer reloadMu.Unlock()

		if renderer == nil {
			writeLazyDevControlResponse(w, http.StatusInternalServerError, "reload views: renderer is missing\n")
			return
		}
		if err := renderer.Cache(); err != nil {
			writeLazyDevControlResponse(w, http.StatusInternalServerError, fmt.Sprintf("reload views: %v\n", err))
			return
		}
		writeLazyDevControlResponse(w, http.StatusOK, "reload views ok\n")
	})
	controlPlane.Handle("POST "+lazyDevControlViewsPath, reloadViews)
	controlPlane.Handle("POST "+lazyDevReloadViewsPath, reloadViews)
}

func writeLazyDevControlResponse(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}
