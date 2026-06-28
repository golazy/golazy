//go:build lazydev

package lazytelemetry

import (
	"encoding/json"
	"net/http"

	"golazy.dev/lazycontrolplane"
)

// RegisterLazyDevHandlers registers request-monitoring endpoints on the
// application's lazydev control plane.
func RegisterLazyDevHandlers(controlPlane *lazycontrolplane.ControlPlane) {
	if controlPlane == nil {
		return
	}
	controlPlane.Handle("GET "+LazyDevRequestMonitoringPath, http.HandlerFunc(handleRequestMonitoring))
	controlPlane.Handle("POST "+LazyDevRequestMonitoringOnPath, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		SetRequestMonitoringEnabled(true)
		handleRequestMonitoring(w, r)
	}))
	controlPlane.Handle("POST "+LazyDevRequestMonitoringOffPath, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		SetRequestMonitoringEnabled(false)
		handleRequestMonitoring(w, r)
	}))
}

func handleRequestMonitoring(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(RequestMonitoringState())
}
