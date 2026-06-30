package lazypwa

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golazy.dev/lazyassets"
	"golazy.dev/lazyworkers"
)

func TestPWARegistersManifestClientWorkerAndCacheManifest(t *testing.T) {
	assets := lazyassets.New()
	if err := assets.Add("/styles.css", []byte("body{}"), lazyassets.ContentType("text/css")); err != nil {
		t.Fatalf("add asset: %v", err)
	}
	workers := lazyworkers.New()
	app, err := New(Config{
		Installable: true,
		Version:     "v1.2.3",
		Manifest: ManifestConfig{
			Name:       "Example",
			ShortName:  "Example",
			ThemeColor: "#ff0",
			Icons: []Icon{{
				Src:   "/styles.css",
				Sizes: "1x1",
				Type:  "text/css",
			}},
		},
		Offline: OfflineConfig{
			Enabled:       true,
			URLs:          []string{"/"},
			Assets:        []string{"/styles.css"},
			FallbackURL:   "/",
			IncludeAssets: true,
		},
	}, WithAssets(assets), WithWorkers(workers))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if !app.Enabled() {
		t.Fatal("PWA app is disabled")
	}
	if _, ok := workers.Worker(defaultServiceWorkerName); !ok {
		t.Fatal("PWA service worker was not registered")
	}
	if _, err := assets.Path(defaultClientAssetPath); err != nil {
		t.Fatalf("PWA client asset missing: %v", err)
	}

	manifest := getPWAPath(t, app, defaultManifestPath)
	var doc map[string]any
	if err := json.Unmarshal([]byte(manifest), &doc); err != nil {
		t.Fatalf("manifest JSON: %v\n%s", err, manifest)
	}
	if doc["name"] != "Example" || doc["display"] != "standalone" {
		t.Fatalf("manifest = %#v", doc)
	}

	cacheManifest := getPWAPath(t, app, defaultCacheManifestPath)
	if !strings.Contains(cacheManifest, `"version": "v1.2.3"`) || !strings.Contains(cacheManifest, `/styles-`) {
		t.Fatalf("cache manifest missing version or resolved asset:\n%s", cacheManifest)
	}

	worker := getWorkerPath(t, workers, defaultServiceWorkerPath)
	if !strings.Contains(worker, `LAZYPWA_VERSION = "v1.2.3"`) || !strings.Contains(worker, defaultCacheManifestPath) {
		t.Fatalf("worker script missing version/cache manifest:\n%s", worker)
	}
}

func getPWAPath(t *testing.T, app *App, path string) string {
	t.Helper()
	response := httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
	if response.Code != http.StatusOK {
		t.Fatalf("%s status = %d, want 200", path, response.Code)
	}
	return response.Body.String()
}

func getWorkerPath(t *testing.T, workers *lazyworkers.Registry, path string) string {
	t.Helper()
	response := httptest.NewRecorder()
	workers.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
	if response.Code != http.StatusOK {
		t.Fatalf("%s status = %d, want 200", path, response.Code)
	}
	return response.Body.String()
}
