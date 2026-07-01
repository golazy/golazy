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
	"time"

	"golazy.dev/lazyassets"
	"golazy.dev/lazybuildinfo"
	"golazy.dev/lazycache"
	"golazy.dev/lazycontroller"
	"golazy.dev/lazydeps"
	"golazy.dev/lazyfiles"
	filesjsonl "golazy.dev/lazyfiles/jsonl"
	"golazy.dev/lazymedia"
	mediajsonl "golazy.dev/lazymedia/jsonl"
	"golazy.dev/lazyroutes"
	"golazy.dev/lazystorage"
	"golazy.dev/lazytelemetry"
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
		lazybuildinfo.LazyDevBuildInfoPath,
		lazyassets.LazyDevAssetsPath,
		lazydeps.LazyDevDependenciesPath,
		lazycache.LazyDevCachePath,
		lazycache.LazyDevCacheEventsPath,
		lazycache.LazyDevCacheOnPath,
		lazycache.LazyDevCacheOffPath,
		lazymedia.LazyDevMediaPath,
		lazymedia.LazyDevMediaDownloadPath,
		lazymedia.LazyDevMediaStorageUploadPath,
		lazymedia.LazyDevMediaStorageDeletePath,
		lazymedia.LazyDevMediaFileUploadPath,
		lazymedia.LazyDevMediaFileDeletePath,
		lazymedia.LazyDevMediaVariantDeletePath,
		lazytelemetry.LazyDevRequestMonitoringPath,
		lazytelemetry.LazyDevRequestMonitoringOnPath,
		lazytelemetry.LazyDevRequestMonitoringOffPath,
		lazytelemetry.LazyDevRequestTracesPath,
		lazytelemetry.LazyDevRequestTracesClearPath,
	} {
		if !app.ControlPlane.HandlesPath(path) {
			t.Fatalf("control plane does not handle %s", path)
		}
	}
}

func TestLazyDevControlPlaneServesMediaInspector(t *testing.T) {
	dir := t.TempDir()
	fileRepo, err := filesjsonl.New(filepath.Join(dir, "files.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	variantRepo, err := mediajsonl.New(filepath.Join(dir, "variants.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	files := &lazyfiles.Files{
		Repository: fileRepo,
		Storages: map[string]lazystorage.Storage{
			"local": lazystorage.NewFilesystem(filepath.Join(dir, "objects"), lazystorage.WithBaseURL("https://cdn.example.test/files")),
		},
		DefaultStorage: "local",
	}
	file, _, err := files.Put(
		context.Background(),
		strings.NewReader("avatar"),
		lazyfiles.Filename{Name: "avatar.png"},
		lazyfiles.ObjectKey{Key: "uploads/avatar.png"},
		lazystorage.ContentType{Value: "image/png"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := variantRepo.SaveVariant(context.Background(), lazymedia.Variant{
		SourceFileID: file.ID,
		VariantKey:   "thumb",
		OutputFileID: file.ID,
		Status:       lazymedia.StatusReady,
	}); err != nil {
		t.Fatal(err)
	}
	app := New(Config{
		Name:  "test",
		Files: files,
		Media: &lazymedia.Media{Repository: variantRepo},
	})

	response := httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, lazymedia.LazyDevMediaPath+"?storage=local", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("media status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	var snapshot lazymedia.LazyDevSnapshot
	if err := json.Unmarshal(response.Body.Bytes(), &snapshot); err != nil {
		t.Fatalf("decode media response: %v\n%s", err, response.Body.String())
	}
	if len(snapshot.Storages) != 1 || snapshot.Storages[0].Name != "local" || !snapshot.Storages[0].Writable {
		t.Fatalf("storages = %#v, want writable local", snapshot.Storages)
	}
	if len(snapshot.StorageObjects) != 1 || snapshot.StorageObjects[0].Key != "uploads/avatar.png" {
		t.Fatalf("storage objects = %#v, want uploaded avatar", snapshot.StorageObjects)
	}
	if len(snapshot.Files) != 1 || snapshot.Files[0].ID != file.ID || len(snapshot.Files[0].Variants) != 1 {
		t.Fatalf("files = %#v, want file with variant", snapshot.Files)
	}
	if snapshot.Files[0].Variants[0].VariantKey != "thumb" || snapshot.Files[0].Variants[0].OutputURL == "" {
		t.Fatalf("variants = %#v, want thumb with output URL", snapshot.Files[0].Variants)
	}
}

func TestLazyDevControlPlaneServesDependencies(t *testing.T) {
	app := New(Config{
		Name: "test",
		Dependencies: func(scope *lazydeps.Scope) error {
			db, err := lazydeps.Service(scope, "db", func(ctx context.Context) (context.Context, string, error, context.CancelFunc) {
				return ctx, "db", nil, nil
			})
			if err != nil {
				return err
			}
			_, err = lazydeps.Service(scope, "posts", func(ctx context.Context) (context.Context, string, error, context.CancelFunc) {
				_ = db.Use()
				return ctx, "posts", nil, nil
			})
			return err
		},
	})
	if app.ControlPlane == nil {
		t.Fatal("lazydev app did not install a control plane")
	}

	response := httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, lazydeps.LazyDevDependenciesPath, nil))
	if response.Code != http.StatusOK {
		t.Fatalf("dependencies status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if got := response.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("dependencies Content-Type = %q, want JSON", got)
	}

	var graph lazydeps.Graph
	if err := json.Unmarshal(response.Body.Bytes(), &graph); err != nil {
		t.Fatalf("decode dependencies response: %v\n%s", err, response.Body.String())
	}
	wantEdges := []lazydeps.Edge{
		{From: "app", To: "db"},
		{From: "app", To: "posts"},
		{From: "posts", To: "db"},
	}
	if strings.Join(graph.Nodes, ",") != "app,db,posts" {
		t.Fatalf("nodes = %#v, want app,db,posts", graph.Nodes)
	}
	if len(graph.Edges) != len(wantEdges) {
		t.Fatalf("edges = %#v, want %#v", graph.Edges, wantEdges)
	}
	for index, want := range wantEdges {
		if graph.Edges[index] != want {
			t.Fatalf("edges[%d] = %#v, want %#v", index, graph.Edges[index], want)
		}
	}
}

func TestLazyDevDependencyShutdownMarksAppNotReady(t *testing.T) {
	app := New(Config{Name: "test"})
	if app.ControlPlane == nil {
		t.Fatal("lazydev app did not install a control plane")
	}

	ready := httptest.NewRecorder()
	app.ServeHTTP(ready, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if ready.Code != http.StatusOK {
		t.Fatalf("readyz before shutdown = %d, want %d: %s", ready.Code, http.StatusOK, ready.Body.String())
	}

	response := httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodPost, lazydeps.LazyDevDependencyShutdownPath, nil))
	if response.Code != http.StatusOK {
		t.Fatalf("shutdown status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}

	var state lazydeps.LazyDevShutdownState
	deadline := time.Now().Add(2 * time.Second)
	for {
		stateResponse := httptest.NewRecorder()
		app.ServeHTTP(stateResponse, httptest.NewRequest(http.MethodGet, lazydeps.LazyDevDependencyShutdownPath, nil))
		if stateResponse.Code != http.StatusOK {
			t.Fatalf("shutdown state status = %d, want %d: %s", stateResponse.Code, http.StatusOK, stateResponse.Body.String())
		}
		if err := json.Unmarshal(stateResponse.Body.Bytes(), &state); err != nil {
			t.Fatalf("decode shutdown state: %v\n%s", err, stateResponse.Body.String())
		}
		if !state.Ready && state.ReadyStatus == http.StatusServiceUnavailable {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("shutdown state never became not ready: %#v", state)
		}
		time.Sleep(10 * time.Millisecond)
	}

	ready = httptest.NewRecorder()
	app.ServeHTTP(ready, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if ready.Code != http.StatusServiceUnavailable {
		t.Fatalf("readyz after shutdown = %d, want %d: %s", ready.Code, http.StatusServiceUnavailable, ready.Body.String())
	}
	if state.ActiveRequests != 0 {
		t.Fatalf("active requests = %d, want 0", state.ActiveRequests)
	}
}

func TestLazyDevControlPlaneServesRequestMonitoringToggle(t *testing.T) {
	lazytelemetry.SetRequestMonitoringEnabled(false)
	t.Cleanup(func() {
		lazytelemetry.SetRequestMonitoringEnabled(false)
	})

	app := New(Config{Name: "test"})
	if app.ControlPlane == nil {
		t.Fatal("lazydev app did not install a control plane")
	}

	got := requestLazyAppDevRequestMonitoring(t, app, http.MethodGet, lazytelemetry.LazyDevRequestMonitoringPath)
	if got.Enabled {
		t.Fatal("request monitoring enabled = true, want default false")
	}
	if got.Directory != ".tmp/traces" {
		t.Fatalf("request monitoring directory = %q, want .tmp/traces", got.Directory)
	}

	got = requestLazyAppDevRequestMonitoring(t, app, http.MethodPost, lazytelemetry.LazyDevRequestMonitoringOnPath)
	if !got.Enabled {
		t.Fatal("request monitoring enabled = false after on")
	}

	got = requestLazyAppDevRequestMonitoring(t, app, http.MethodPost, lazytelemetry.LazyDevRequestMonitoringOffPath)
	if got.Enabled {
		t.Fatal("request monitoring enabled = true after off")
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

type lazyDevRequestMonitoringTestResponse struct {
	Enabled   bool   `json:"enabled"`
	Directory string `json:"directory"`
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

func requestLazyAppDevRequestMonitoring(t *testing.T, app *App, method string, path string) lazyDevRequestMonitoringTestResponse {
	t.Helper()
	response := httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(method, path, nil))
	if response.Code != http.StatusOK {
		t.Fatalf("%s %s status = %d, want %d: %s", method, path, response.Code, http.StatusOK, response.Body.String())
	}
	if got := response.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("request monitoring Content-Type = %q, want JSON", got)
	}
	var out lazyDevRequestMonitoringTestResponse
	if err := json.Unmarshal(response.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode request monitoring response: %v\n%s", err, response.Body.String())
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
