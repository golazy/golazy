package lazydispatch

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddleware_Empty(t *testing.T) {
	d := New()

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	d.ServeHTTP(w, r)

	expect2(t, d, "GET", "/", nil, http.StatusNotFound, "Not Found")

	// With at least one middleware
	d.Use(func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h.ServeHTTP(w, r)
		})
	})

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/", nil)
	d.ServeHTTP(w, r)
	expect2(t, d, "GET", "/", nil, http.StatusNotFound, "Not Found")

}

func expect2(t *testing.T, h http.Handler, verb, path string, body io.Reader, expCode int, expBody string) {
	t.Helper()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(verb, path, body)
	h.ServeHTTP(w, r)
	if w.Code != expCode {
		t.Errorf("expected code %d, got %d", expCode, w.Code)
	}
	if w.Body.String() != expBody {
		t.Errorf("expected body %q, got %q", body, w.Body.String())
	}
}
