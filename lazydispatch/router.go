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

// RouteOnly applies middlewares only to requests handled by router.
func RouteOnly(router RouteHandler, middlewares ...Middleware) Middleware {
	return MiddlewareFunc(func(next http.Handler) http.Handler {
		if next == nil {
			next = http.NotFoundHandler()
		}
		routed := next
		for i := len(middlewares) - 1; i >= 0; i-- {
			routed = instrumentMiddleware(middlewares[i], routed, i)
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if router != nil && router.HandlesPath(r.URL.Path) {
				routed.ServeHTTP(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	})
}
