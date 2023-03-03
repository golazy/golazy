package lazyapp

import (
	"net/http"
	"strings"
)

type mounts map[string]http.Handler

func (m mounts) Middleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for prefix, handler := range m {
			if strings.HasPrefix(r.URL.Path+"/", prefix) ||
				r.URL.Path == prefix {

				r.URL.Path = "/" + strings.TrimPrefix(r.URL.Path, prefix)
				handler.ServeHTTP(w, r)
				return
			}
		}
		h.ServeHTTP(w, r)
	})
}
