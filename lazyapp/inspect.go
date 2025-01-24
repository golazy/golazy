//go:build lazy

package lazyapp

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"

	"golazy.dev/lazyassets"
	"golazy.dev/lazycontext"
	"golazy.dev/lazydispatch"
	"golazy.dev/lazyview"
)

func init() {
	defaultMiddlewares = append(defaultMiddlewares, ExposeViews, ExposePublic, ExposeRoutes)
}

func ExposeViews(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/_inspect/views.json" {
			h.ServeHTTP(w, r)
			return
		}
		views := lazycontext.Get[*lazyview.Views](r.Context())
		paths := []string{}
		fs.WalkDir(views.FS, ".", func(path string, d fs.DirEntry, err error) error {
			if d.IsDir() {
				return nil
			}
			paths = append(paths, path)
			return nil
		})
		err := json.NewEncoder(w).Encode(paths)
		if err != nil {
			panic(err)
		}
	})
}

func ExposePublic(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/_inspect/assets.json") {
			h.ServeHTTP(w, r)
			return
		}
		assets := lazycontext.Get[*lazyassets.Storage](r.Context())
		err := json.NewEncoder(w).Encode(assets.Files)
		if err != nil {
			panic(err)
		}

	})
}

func ExposeRoutes(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/_inspect/routes.json" {
			h.ServeHTTP(w, r)
			return
		}
		dispatcher := lazycontext.Get[*lazydispatch.Dispatcher](r.Context())
		json.NewEncoder(w).Encode(dispatcher.Routes)
	})
}
