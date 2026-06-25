//go:build lazydev

package lazyapp

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golazy.dev/lazycache"
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
	app.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/views", nil))
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

	response = httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/_golazy/views/reload", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("legacy reload status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
}

func TestLazyDevControlPlaneAggregatesPackageHandlers(t *testing.T) {
	app := New(Config{Name: "test"})
	if app.ControlPlane == nil {
		t.Fatal("lazydev app did not install a control plane")
	}

	for _, path := range []string{
		lazyDevControlViewsPath,
		lazyDevReloadViewsPath,
		lazyroutes.LazyDevRoutesPath,
		lazycontroller.LazyDevOpenEditorPath,
		lazycache.LazyDevCachePath,
		lazycache.LazyDevCacheOnPath,
		lazycache.LazyDevCacheOffPath,
	} {
		if !app.ControlPlane.HandlesPath(path) {
			t.Fatalf("control plane does not handle %s", path)
		}
	}
}

func TestLazyDevControlPlaneServesCache(t *testing.T) {
	app := New(Config{Name: "test"})
	if app.ControlPlane == nil {
		t.Fatal("lazydev app did not install a control plane")
	}
	if err := app.Cache.Set("Ada", "users", 1); err != nil {
		t.Fatal(err)
	}

	got := requestLazyAppDevCache(t, app, http.MethodGet, lazycache.LazyDevCachePath)
	if !got.Enabled {
		t.Fatal("cache enabled = false, want true")
	}
	if got.Stats.Entries != 1 || got.Stats.Sets != 1 {
		t.Fatalf("cache stats = %#v, want entries=1 sets=1", got.Stats)
	}
	if len(got.Keys) != 1 || got.Keys[0] != "users-1" {
		t.Fatalf("cache keys = %#v, want [users-1]", got.Keys)
	}

	got = requestLazyAppDevCache(t, app, http.MethodPost, lazycache.LazyDevCacheOffPath)
	if got.Enabled {
		t.Fatal("cache enabled = true after off")
	}
	if _, err := app.Cache.Get("users", 1); !errors.Is(err, lazycache.ErrMiss) {
		t.Fatalf("Get while disabled error = %v, want ErrMiss", err)
	}

	got = requestLazyAppDevCache(t, app, http.MethodPost, lazycache.LazyDevCacheOnPath)
	if !got.Enabled {
		t.Fatal("cache enabled = false after on")
	}
	if value, err := lazycache.Get[string](app.Cache, "users", 1); err != nil || value != "Ada" {
		t.Fatalf("Get after on = %q, %v; want Ada, nil", value, err)
	}
}

func TestLazyDevControlPlaneServesRoutes(t *testing.T) {
	app := New(Config{
		Name: "test",
		Drawer: func(router *lazyroutes.Scope) {
			router.Get("/", newLazyDevReloadController, (*lazyDevReloadController).Index)
		},
	})
	if app.ControlPlane == nil {
		t.Fatal("lazydev app did not install a control plane")
	}

	response := httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/routes", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("routes status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if got := response.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("routes Content-Type = %q, want JSON", got)
	}
	if !strings.Contains(response.Body.String(), `"path":"/"`) {
		t.Fatalf("routes body = %s, want root route", response.Body.String())
	}
}

type lazyDevCacheTestResponse struct {
	Enabled bool            `json:"enabled"`
	Stats   lazycache.Stats `json:"stats"`
	Keys    []string        `json:"keys"`
}

func requestLazyAppDevCache(t *testing.T, app *App, method string, path string) lazyDevCacheTestResponse {
	t.Helper()
	response := httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(method, path, nil))
	if response.Code != http.StatusOK {
		t.Fatalf("%s %s status = %d, want %d: %s", method, path, response.Code, http.StatusOK, response.Body.String())
	}
	if got := response.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("cache Content-Type = %q, want JSON", got)
	}
	var out lazyDevCacheTestResponse
	if err := json.Unmarshal(response.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode cache response: %v\n%s", err, response.Body.String())
	}
	return out
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
