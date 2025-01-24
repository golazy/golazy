//go:build lazy
// +build lazy

package lazydispatch

import (
	"encoding/json"
	"net/http"

	"golazy.dev/lazycontext"
)

func init() {
	defaultMiddlewares = append(defaultMiddlewares, ExposeRoutes)
}

func ExposeRoutes(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/_inspect/routes.json" {
			h.ServeHTTP(w, r)
			return
		}
		dispatcher := lazycontext.Get[*Dispatcher](r.Context())
		json.NewEncoder(w).Encode(dispatcher.Routes)
	})
}
