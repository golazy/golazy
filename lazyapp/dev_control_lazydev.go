//go:build lazydev

package lazyapp

import (
	"fmt"
	"net/http"
	"sync"

	"golazy.dev/lazycontroller"
	"golazy.dev/lazycontrolplane"
)

const lazyDevReloadViewsPath = "/_golazy/views/reload"

func lazyDevControlPlane(controlPlane *lazycontrolplane.ControlPlane, renderer *lazycontroller.Renderer) *lazycontrolplane.ControlPlane {
	if controlPlane == nil {
		controlPlane = lazycontrolplane.New(lazycontrolplane.Config{})
	}
	var reloadMu sync.Mutex
	controlPlane.Handle("POST "+lazyDevReloadViewsPath, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
	}))
	return controlPlane
}

func writeLazyDevControlResponse(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}
