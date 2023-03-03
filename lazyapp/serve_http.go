package lazyapp

import (
	"net/http"
	"strings"
)

func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.Init()
	for prefix, handler := range a.mounts {
		if strings.HasPrefix(r.URL.Path, prefix) {
			handler.ServeHTTP(w, r)
			return
		}
	}
	a.h.ServeHTTP(w, r)
}
