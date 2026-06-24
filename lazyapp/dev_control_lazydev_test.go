//go:build lazydev

package lazyapp

import (
	"context"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"golazy.dev/lazycontroller"
	"golazy.dev/lazyroutes"
)

type lazyDevReloadController struct {
	lazycontroller.Base
}

func newLazyDevReloadController(ctx context.Context) (*lazyDevReloadController, error) {
	base, err := lazycontroller.NewBase(ctx, "pages")
	if err != nil {
		return nil, err
	}
	return &lazyDevReloadController{Base: base}, nil
}

func (c *lazyDevReloadController) Index() error {
	return nil
}

func TestLazyDevControlPlaneReloadsViewsWithoutRebuildingApp(t *testing.T) {
	dir := t.TempDir()
	writeLazyDevControlFile(t, filepath.Join(dir, "layouts", "app.html.tpl"), `{{.content}}`)
	viewFile := filepath.Join(dir, "pages", "index.html.tpl")
	writeLazyDevControlFile(t, viewFile, `before`)

	previous := ViewsPath
	ViewsPath = dir
	t.Cleanup(func() {
		ViewsPath = previous
	})

	app := New(Config{
		Name: "test",
		Views: func() (fs.FS, error) {
			return nil, nil
		},
		Drawer: func(router *lazyroutes.Scope) {
			router.Get("/", newLazyDevReloadController, (*lazyDevReloadController).Index)
		},
	})
	if app.ControlPlane == nil {
		t.Fatal("lazydev app did not install a control plane")
	}

	assertLazyDevReloadBody(t, app, "before")
	writeLazyDevControlFile(t, viewFile, `after`)
	assertLazyDevReloadBody(t, app, "before")

	response := httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/_golazy/views/reload", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("reload status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if got, want := response.Body.String(), "reload views ok\n"; got != want {
		t.Fatalf("reload body = %q, want %q", got, want)
	}
	if got := response.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("reload Cache-Control = %q, want no-store", got)
	}
	assertLazyDevReloadBody(t, app, "after")
}

func assertLazyDevReloadBody(t *testing.T, app *App, want string) {
	t.Helper()
	response := httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("page status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if got := response.Body.String(); got != want {
		t.Fatalf("page body = %q, want %q", got, want)
	}
}

func writeLazyDevControlFile(t *testing.T, filename string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
