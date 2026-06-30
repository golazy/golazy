//go:build lazydev

package lazypwa

import (
	"encoding/json"
	"fmt"
	"net/http"

	"golazy.dev/lazycontrolplane"
)

// LazyDevPWAPath is the app control-plane path for PWA state.
const LazyDevPWAPath = "/pwa"

// RegisterLazyDevHandlers registers PWA inspection endpoints.
func RegisterLazyDevHandlers(controlPlane *lazycontrolplane.ControlPlane, app *App) {
	if controlPlane == nil {
		return
	}
	controlPlane.Handle("GET "+LazyDevPWAPath, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		state := State{}
		if app != nil {
			state = app.State()
		}
		if err := json.NewEncoder(w).Encode(state); err != nil {
			http.Error(w, fmt.Sprintf("pwa: %v", err), http.StatusInternalServerError)
		}
	}))
}
