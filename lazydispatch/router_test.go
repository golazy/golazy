package lazydispatch

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type testRouter struct {
	paths map[string]bool
}

func (r testRouter) HandlesPath(path string) bool {
	return r.paths[path]
}

func (r testRouter) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	_, _ = w.Write([]byte("router"))
}

func TestRouterMiddlewareDispatchesKnownPaths(t *testing.T) {
	handler := Router(testRouter{paths: map[string]bool{"/known": true}}).Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("next"))
	}))

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/known", nil))
	if response.Body.String() != "router" {
		t.Fatalf("body = %q, want %q", response.Body.String(), "router")
	}
}

func TestRouterMiddlewareFallsThroughUnknownPaths(t *testing.T) {
	handler := Router(testRouter{paths: map[string]bool{"/known": true}}).Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("next"))
	}))

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/missing", nil))
	if response.Body.String() != "next" {
		t.Fatalf("body = %q, want %q", response.Body.String(), "next")
	}
}
