package routes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewInstallsPublicFallback(t *testing.T) {
	public := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	mux := New(WithPublic(context.Background(), public))

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/asset.css", nil))
	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
}

func TestPublicFallbackRejectsUnsupportedMethods(t *testing.T) {
	public := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("public handler should not be called")
	})
	mux := New(WithPublic(context.Background(), public))

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/asset.css", nil))
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusMethodNotAllowed)
	}
	if response.Header().Get("Allow") != http.MethodGet {
		t.Fatalf("Allow = %q, want %q", response.Header().Get("Allow"), http.MethodGet)
	}
}
