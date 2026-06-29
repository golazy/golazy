package lazydispatch

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"golazy.dev/lazysse"
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

func TestResponseBufferCanStartStreaming(t *testing.T) {
	handler := ResponseBuffer().Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Test", "stream")
		_, _ = fmt.Fprint(w, "discarded")
		stream, err := lazysse.Start(w, r)
		if err != nil {
			t.Fatal(err)
		}
		defer stream.Close()
		if err := stream.Send(lazysse.Event{Event: "ready", Data: []string{"ok"}}); err != nil {
			t.Fatal(err)
		}
		w.(interface{ Reset() }).Reset()
		_, _ = fmt.Fprint(w, "late error body")
	}))

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if got, want := response.Header().Get("X-Test"), "stream"; got != want {
		t.Fatalf("X-Test = %q, want %q", got, want)
	}
	if got, want := response.Body.String(), "event: ready\ndata: ok\n\n"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestPooledResponseBufferClearsState(t *testing.T) {
	first := AcquireBufferedResponseWriter(httptest.NewRecorder())
	first.Header().Set("X-Test", "stale")
	first.WriteHeader(http.StatusCreated)
	_, _ = fmt.Fprint(first, "stale")
	ReleaseBufferedResponseWriter(first)

	second := AcquireBufferedResponseWriter(httptest.NewRecorder())
	defer ReleaseBufferedResponseWriter(second)
	if got := second.Header().Get("X-Test"); got != "" {
		t.Fatalf("pooled header = %q, want empty", got)
	}
	if second.WasResponseSent() {
		t.Fatal("pooled response was marked sent")
	}
	if got := second.body.Len(); got != 0 {
		t.Fatalf("pooled body length = %d, want 0", got)
	}
}
