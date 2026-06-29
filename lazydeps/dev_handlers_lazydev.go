//go:build lazydev

package lazydeps

import (
	"encoding/json"
	"fmt"
	"net/http"

	"golazy.dev/lazycontrolplane"
)

const LazyDevDependenciesPath = "/dependencies"

// RegisterLazyDevHandlers registers dependency graph inspection endpoints.
func RegisterLazyDevHandlers(controlPlane *lazycontrolplane.ControlPlane, dependencies *Scope) {
	if controlPlane == nil {
		return
	}
	controlPlane.Handle("GET "+LazyDevDependenciesPath, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		graph := Graph{Nodes: []string{}, Edges: []Edge{}}
		if dependencies != nil {
			graph = dependencies.Graph()
		}
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if err := json.NewEncoder(w).Encode(graph); err != nil {
			http.Error(w, fmt.Sprintf("dependencies: %v", err), http.StatusInternalServerError)
		}
	}))
}
