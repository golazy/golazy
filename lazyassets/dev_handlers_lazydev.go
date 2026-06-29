//go:build lazydev

package lazyassets

import (
	"encoding/json"
	"fmt"
	"net/http"

	"golazy.dev/lazycontrolplane"
)

const LazyDevAssetsPath = "/assets"

// RegisterLazyDevHandlers registers asset inventory endpoints.
func RegisterLazyDevHandlers(controlPlane *lazycontrolplane.ControlPlane, registry *Registry) {
	if controlPlane == nil {
		return
	}
	controlPlane.Handle("GET "+LazyDevAssetsPath, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if registry == nil {
			if err := json.NewEncoder(w).Encode(Manifest{}); err != nil {
				http.Error(w, fmt.Sprintf("assets: %v", err), http.StatusInternalServerError)
			}
			return
		}
		if err := json.NewEncoder(w).Encode(registry.Manifest()); err != nil {
			http.Error(w, fmt.Sprintf("assets: %v", err), http.StatusInternalServerError)
		}
	}))
}
