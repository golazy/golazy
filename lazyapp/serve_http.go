package lazyapp

import (
	"net/http"
)

func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.h.ServeHTTP(w, r)
}
