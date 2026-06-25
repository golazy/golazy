//go:build lazydev

package lazyroutes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golazy.dev/lazycontrolplane"
)

func TestRegisterLazyDevHandlersServesRoutes(t *testing.T) {
	router := New(context.Background())
	router.HandleFunc(http.MethodGet, "/", func(http.ResponseWriter, *http.Request) error {
		return nil
	})
	controlPlane := lazycontrolplane.New(lazycontrolplane.Config{})
	RegisterLazyDevHandlers(controlPlane, router)

	response := httptest.NewRecorder()
	controlPlane.ServeHTTP(response, httptest.NewRequest(http.MethodGet, LazyDevRoutesPath, nil))

	if response.Code != http.StatusOK {
		t.Fatalf("routes status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if got := response.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("routes Content-Type = %q, want JSON", got)
	}
	if !strings.Contains(response.Body.String(), `"path":"/"`) {
		t.Fatalf("routes body = %s, want root route", response.Body.String())
	}
}
