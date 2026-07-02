package lazyapp

import (
	"net/http"
	"strings"

	"golazy.dev/lazymcp"
	"golazy.dev/lazyoauth"
)

type mcpMiddleware struct {
	oauth *lazyoauth.Server
	mcp   *lazymcp.Scope
}

func (mcpMiddleware) MiddlewareName() string {
	return "lazymcp.Scope"
}

func (middleware mcpMiddleware) Handler(next http.Handler) http.Handler {
	if next == nil {
		next = http.NotFoundHandler()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/mcp" {
			handler := middleware.mcp.Handler(next)
			if middleware.oauth != nil {
				handler = middleware.oauth.Protect(handler)
			}
			handler.ServeHTTP(w, r)
			return
		}
		if middleware.oauth != nil && middleware.oauth.HandlesPath(r.URL.Path) {
			middleware.oauth.ServeHTTP(w, r)
			return
		}
		if middleware.oauth != nil && strings.HasPrefix(r.URL.Path, "/.well-known/oauth-") {
			middleware.oauth.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}
