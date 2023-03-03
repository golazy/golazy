package lazyapp

import (
	"fmt"
	"net/http"
)

func panicMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.WriteHeader(500)
				w.Write([]byte(fmt.Sprintf("Internal Server Error: %s", err)))
			}
		}()
		h.ServeHTTP(w, r)
	})
}
