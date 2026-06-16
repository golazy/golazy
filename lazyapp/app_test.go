package lazyapp

import (
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"golazy.dev/lazyroutes"
)

func TestAppAddsETagToRoutesOnly(t *testing.T) {
	app := New(Config{
		Name: "test",
		Drawer: func(router *lazyroutes.Scope) {
			router.HandleFunc(http.MethodGet, "/", func(w http.ResponseWriter, _ *http.Request) error {
				_, _ = fmt.Fprint(w, "home")
				return nil
			})
		},
		Public: func() (fs.FS, error) {
			return fstest.MapFS{
				"styles.css": {Data: []byte("body { color: black; }")},
			}, nil
		},
	})

	response := httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("route status = %d, want %d", response.Code, http.StatusOK)
	}
	if response.Header().Get("ETag") == "" {
		t.Fatal("route ETag is empty")
	}

	response = httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/styles.css", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("public status = %d, want %d", response.Code, http.StatusOK)
	}
	if response.Header().Get("ETag") != "" {
		t.Fatalf("public ETag = %q, want empty", response.Header().Get("ETag"))
	}
}
