//go:build lazydev

package lazyapp

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"golazy.dev/lazycache"
	"golazy.dev/lazycontroller"
	"golazy.dev/lazycontrolplane"
	"golazy.dev/lazyroutes"
)

const lazyDevReloadViewsPath = "/_golazy/views/reload"
const lazyDevControlViewsPath = "/views"

func lazyDevContext(ctx context.Context) context.Context {
	return lazycontroller.LazyDevContext(ctx)
}

func lazyDevControlPlane(controlPlane *lazycontrolplane.ControlPlane, renderer *lazycontroller.Renderer, router *lazyroutes.Scope, cache *lazycache.Cache) *lazycontrolplane.ControlPlane {
	if controlPlane == nil {
		controlPlane = lazycontrolplane.New(lazycontrolplane.Config{})
	}
	registerLazyDevViewHandlers(controlPlane, renderer)
	lazyroutes.RegisterLazyDevHandlers(controlPlane, router)
	lazycontroller.RegisterLazyDevHandlers(controlPlane)
	lazycache.RegisterLazyDevHandlers(controlPlane, cache)
	return controlPlane
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
