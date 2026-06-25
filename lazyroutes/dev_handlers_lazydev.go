//go:build lazydev

package lazyroutes

import (
	"encoding/json"
	"fmt"
	"net/http"

	"golazy.dev/lazycontrolplane"
)

const LazyDevRoutesPath = "/routes"

func RegisterLazyDevHandlers(controlPlane *lazycontrolplane.ControlPlane, router *Scope) {
	if controlPlane == nil {
		return
	}
	controlPlane.Handle("GET "+LazyDevRoutesPath, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		routes := RouteTable(nil)
		if router != nil {
			routes = append(routes, router.Routes...)
		}
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if err := json.NewEncoder(w).Encode(routes); err != nil {
			http.Error(w, fmt.Sprintf("routes: %v", err), http.StatusInternalServerError)
		}
	}))
}
