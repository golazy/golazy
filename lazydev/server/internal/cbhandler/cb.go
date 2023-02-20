package cbhandler

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

type Handler struct {
	fallback http.Handler
	origin   http.Handler
}

func New(fallbackHandler http.Handler) *Handler {
	return &Handler{
		fallback: fallbackHandler,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.origin != nil {
		h.origin.ServeHTTP(w, r)
		return
	}
	h.fallback.ServeHTTP(w, r)
}

func (h *Handler) Open(url *url.URL) {
	h.origin = httputil.NewSingleHostReverseProxy(url)
}
func (h *Handler) Close() {
	h.origin = nil
}
