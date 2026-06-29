//go:build lazydev

package lazybuildinfo

import (
	"encoding/json"
	"fmt"
	"net/http"

	"golazy.dev/lazycontrolplane"
)

// LazyDevBuildInfoPath is the lazydev control-plane path for Go build
// metadata.
//
// GET requests return a no-store JSON snapshot from
// runtime/debug.ReadBuildInfo. The response includes whether build information
// was available, the Go version, the main package path, the main module,
// dependencies, module replacements, and build settings.
const LazyDevBuildInfoPath = "/buildinfo"

// RegisterLazyDevHandlers registers lazydev build metadata endpoints on
// controlPlane.
//
// This function exists only in lazydev builds. lazyapp normally calls it while
// aggregating package-owned development handlers onto the application's
// lazycontrolplane.ControlPlane, so applications that use lazyapp do not need to
// call it directly. Custom development servers can call it when they create and
// serve their own control plane.
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
