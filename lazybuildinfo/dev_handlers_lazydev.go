//go:build lazydev

package lazybuildinfo

import (
	"encoding/json"
	"fmt"
	"net/http"

	"golazy.dev/lazycontrolplane"
)

const LazyDevBuildInfoPath = "/buildinfo"

// RegisterLazyDevHandlers registers build metadata endpoints.
func RegisterLazyDevHandlers(controlPlane *lazycontrolplane.ControlPlane) {
	if controlPlane == nil {
		return
	}
	controlPlane.Handle("GET "+LazyDevBuildInfoPath, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if err := json.NewEncoder(w).Encode(snapshot()); err != nil {
			http.Error(w, fmt.Sprintf("buildinfo: %v", err), http.StatusInternalServerError)
		}
	}))
}
