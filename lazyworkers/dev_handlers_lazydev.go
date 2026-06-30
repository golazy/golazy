//go:build lazydev

package lazyworkers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"golazy.dev/lazycontrolplane"
)

// LazyDevWorkersPath is the app control-plane path for worker inventory.
const LazyDevWorkersPath = "/workers"

// RegisterLazyDevHandlers registers worker inspection endpoints.
func RegisterLazyDevHandlers(controlPlane *lazycontrolplane.ControlPlane, registry *Registry) {
	if controlPlane == nil {
		return
	}
	controlPlane.Handle("GET "+LazyDevWorkersPath, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if registry == nil {
			if err := json.NewEncoder(w).Encode(Manifest{}); err != nil {
				http.Error(w, fmt.Sprintf("workers: %v", err), http.StatusInternalServerError)
			}
			return
		}
		if err := json.NewEncoder(w).Encode(registry.Manifest()); err != nil {
			http.Error(w, fmt.Sprintf("workers: %v", err), http.StatusInternalServerError)
		}
	}))
}
