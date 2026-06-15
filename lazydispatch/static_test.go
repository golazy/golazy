package lazydispatch

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestStaticServesExistingFiles(t *testing.T) {
	handler := Static(fstest.MapFS{
		"styles.css": {Data: []byte("body { color: black; }")},
	}).Handler(http.NotFoundHandler())

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/styles.css", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if response.Body.String() != "body { color: black; }" {
		t.Fatalf("body = %q", response.Body.String())
	}
}

func TestStaticFallsThroughMissingFiles(t *testing.T) {
	handler := Static(fstest.MapFS{}).Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("next"))
	}))

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/missing.txt", nil))
	if response.Body.String() != "next" {
		t.Fatalf("body = %q, want %q", response.Body.String(), "next")
	}
}

func TestStaticRejectsUnsupportedMethodsForExistingFiles(t *testing.T) {
	handler := Static(fstest.MapFS{
		"styles.css": {Data: []byte("body { color: black; }")},
	}).Handler(http.NotFoundHandler())

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/styles.css", nil))
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusMethodNotAllowed)
	}
	if response.Header().Get("Allow") != http.MethodGet {
		t.Fatalf("Allow = %q, want %q", response.Header().Get("Allow"), http.MethodGet)
	}
}
