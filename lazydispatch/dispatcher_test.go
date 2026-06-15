package lazydispatch

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
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
