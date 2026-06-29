package lazydispatch

import "net/http"

type RouteHandler interface {
	http.Handler
	HandlesPath(path string) bool
}

type routerMiddleware struct {
	router RouteHandler
}

func (routerMiddleware) MiddlewareName() string {
	return "lazydispatch.Router"
}

func (middleware routerMiddleware) Handler(next http.Handler) http.Handler {
	if next == nil {
		next = http.NotFoundHandler()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if middleware.router != nil && middleware.router.HandlesPath(r.URL.Path) {
			middleware.router.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type routeOnlyMiddleware struct {
	router      RouteHandler
	middlewares []Middleware
}

func (routeOnlyMiddleware) MiddlewareName() string {
	return "lazydispatch.RouteOnly"
}

func (middleware routeOnlyMiddleware) Handler(next http.Handler) http.Handler {
	if next == nil {
		next = http.NotFoundHandler()
	}
	routed := next
	for i := len(middleware.middlewares) - 1; i >= 0; i-- {
		routed = instrumentMiddleware(middleware.middlewares[i], routed, i)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if middleware.router != nil && middleware.router.HandlesPath(r.URL.Path) {
			routed.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func Router(router RouteHandler) Middleware {
	return routerMiddleware{router: router}
}

// RouteOnly applies middlewares only to requests handled by router.
func RouteOnly(router RouteHandler, middlewares ...Middleware) Middleware {
	return routeOnlyMiddleware{router: router, middlewares: middlewares}
}
