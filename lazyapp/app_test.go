package lazyapp

import (
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"golazy.dev/lazyassets"
	"golazy.dev/lazyroutes"
)

func TestAppAddsDynamicETagToRoutesAndAssetETagToPublicFiles(t *testing.T) {
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
	if response.Header().Get("ETag") == "" {
		t.Fatal("public ETag is empty")
	}
	if got := response.Header().Get("Cache-Control"); got != "public, max-age=0, must-revalidate" {
		t.Fatalf("public Cache-Control = %q, want asset logical cache policy", got)
	}
}

func TestAppRegistersGeneratedAssetSources(t *testing.T) {
	app := New(Config{
		Name: "test",
		Assets: []lazyassets.Source{
			lazyassets.SourceFunc(func(registry *lazyassets.Registry) error {
				return registry.Add("/generated.js", []byte("console.log('generated');"))
			}),
		},
	})

	response := httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/generated.js", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("generated status = %d, want %d", response.Code, http.StatusOK)
	}
	if response.Body.String() != "console.log('generated');" {
		t.Fatalf("generated body = %q, want generated JavaScript", response.Body.String())
	}
	if app.Assets == nil || app.Assets.Empty() {
		t.Fatal("app Assets registry is empty")
	}
}
