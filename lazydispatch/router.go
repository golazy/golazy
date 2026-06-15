package lazydispatch

import "net/http"

type RouteHandler interface {
	http.Handler
	HandlesPath(path string) bool
}

func Router(router RouteHandler) Middleware {
	return MiddlewareFunc(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if router != nil && router.HandlesPath(r.URL.Path) {
				router.ServeHTTP(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	})
}
