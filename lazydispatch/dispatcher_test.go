package lazydispatch

import (
	"context"
	"fmt"
	"log/slog"
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
	if got, ok := spanBoolAttr(children[0], "middleware.next_called"); !ok || !got {
		t.Fatalf("outer next_called = %v, %v; want true, true", got, ok)
	}
	if got, ok := spanBoolAttr(children[0], "middleware.handled"); !ok || got {
		t.Fatalf("outer handled = %v, %v; want false, true", got, ok)
	}
	if got, ok := spanBoolAttr(innerChildren[0], "middleware.next_called"); !ok || !got {
		t.Fatalf("inner next_called = %v, %v; want true, true", got, ok)
	}
}

func TestDispatcherMarksMiddlewareThatDoesNotCallNextAsHandled(t *testing.T) {
	dispatcher := NewDispatcher()
	dispatcher.Use(testMiddleware{name: "outer"})
	dispatcher.Use(stopMiddleware{name: "stop"})

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
	stopChildren := children[0].Children()
	if len(stopChildren) != 1 {
		t.Fatalf("outer children = %d, want 1", len(stopChildren))
	}
	if got, want := stopChildren[0].Name(), "middleware stop"; got != want {
		t.Fatalf("stop span name = %q, want %q", got, want)
	}
	if got, ok := spanBoolAttr(stopChildren[0], "middleware.next_called"); !ok || got {
		t.Fatalf("stop next_called = %v, %v; want false, true", got, ok)
	}
	if got, ok := spanBoolAttr(stopChildren[0], "middleware.handled"); !ok || !got {
		t.Fatalf("stop handled = %v, %v; want true, true", got, ok)
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

type stopMiddleware struct {
	name string
}

func (m stopMiddleware) MiddlewareName() string {
	return m.name
}

func (m stopMiddleware) Handler(http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})
}

func spanBoolAttr(span *lazytracing.Span, name string) (bool, bool) {
	for _, attr := range span.Attributes() {
		if attr.Key == name && attr.Value.Kind() == slog.KindBool {
			return attr.Value.Bool(), true
		}
	}
	return false, false
}
