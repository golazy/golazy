package lazydispatch

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"golazy.dev/lazytelemetry/lazytracing"
)

func TestDispatcherRunsMiddlewareInUseOrder(t *testing.T) {
	dispatcher := NewDispatcher()
	calls := []string{}

	dispatcher.Use(MiddlewareFunc(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls = append(calls, "before-a")
			next.ServeHTTP(w, r)
			calls = append(calls, "after-a")
		})
	}))
	dispatcher.Use(MiddlewareFunc(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls = append(calls, "before-b")
			next.ServeHTTP(w, r)
			calls = append(calls, "after-b")
		})
	}))

	response := httptest.NewRecorder()
	dispatcher.Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls = append(calls, "handler")
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))

	got := fmt.Sprint(calls)
	want := "[before-a before-b handler after-b after-a]"
	if got != want {
		t.Fatalf("calls = %s, want %s", got, want)
	}
	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
}

func TestDispatcherWrapsMiddlewareInTraceRegions(t *testing.T) {
	dispatcher := NewDispatcher()
	dispatcher.Use(testMiddleware{name: "outer"})
	dispatcher.Use(testMiddleware{name: "inner"})

	ctx, root := lazytracing.StartSpan(context.Background(), "http.server.request")
	defer root.End()
	request := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	response := httptest.NewRecorder()

	dispatcher.Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(response, request)

	children := root.Children()
	if len(children) != 1 {
		t.Fatalf("root children = %d, want 1", len(children))
	}
	if got, want := children[0].Name(), "middleware outer"; got != want {
		t.Fatalf("outer span name = %q, want %q", got, want)
	}
	innerChildren := children[0].Children()
	if len(innerChildren) != 1 {
		t.Fatalf("outer children = %d, want 1", len(innerChildren))
	}
	if got, want := innerChildren[0].Name(), "middleware inner"; got != want {
		t.Fatalf("inner span name = %q, want %q", got, want)
	}
	if innerChildren[0].ParentID() != children[0].SpanID() {
		t.Fatalf("inner parent = %q, want %q", innerChildren[0].ParentID(), children[0].SpanID())
	}
}

type testMiddleware struct {
	name string
}

func (m testMiddleware) MiddlewareName() string {
	return m.name
}

func (m testMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}
