package lazydispatch

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResponseBufferFlushesAfterHandlerReturns(t *testing.T) {
	seen := false
	handler := ResponseBuffer().Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Test", "yes")
		w.WriteHeader(http.StatusCreated)
		_, _ = fmt.Fprint(w, "created")
		seen = w.(interface{ WasResponseSent() bool }).WasResponseSent()
	}))

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))

	if !seen {
		t.Fatal("WasResponseSent inside handler = false, want true")
	}
	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusCreated)
	}
	if response.Body.String() != "created" {
		t.Fatalf("body = %q, want %q", response.Body.String(), "created")
	}
	if response.Header().Get("X-Test") != "yes" {
		t.Fatalf("X-Test = %q, want yes", response.Header().Get("X-Test"))
	}
}

func TestResponseBufferCanResetResponse(t *testing.T) {
	handler := ResponseBuffer().Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Test", "stale")
		w.WriteHeader(http.StatusAccepted)
		_, _ = fmt.Fprint(w, "stale")
		w.(interface{ Reset() }).Reset()
		w.Header().Set("X-Test", "fresh")
		w.WriteHeader(http.StatusTeapot)
		_, _ = fmt.Fprint(w, "fresh")
	}))

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))

	if response.Code != http.StatusTeapot {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusTeapot)
	}
	if response.Body.String() != "fresh" {
		t.Fatalf("body = %q, want %q", response.Body.String(), "fresh")
	}
	if response.Header().Get("X-Test") != "fresh" {
		t.Fatalf("X-Test = %q, want fresh", response.Header().Get("X-Test"))
	}
}
