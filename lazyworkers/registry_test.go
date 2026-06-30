package lazyworkers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golazy.dev/lazyview"
)

func TestRegistryServesGeneratedServiceWorkerAndHelpers(t *testing.T) {
	registry := New()
	if err := registry.AddScript("pwa", ServiceWorker, "/service-worker.js", []byte(`self.version = "test"`),
		WithScope("/docs/"),
		WithDescription("test service worker"),
		WithPWA(),
	); err != nil {
		t.Fatalf("AddScript: %v", err)
	}
	if err := registry.AddAsset("search", WebWorker, "/assets/search-worker.js", WithScriptType(ModuleScript)); err != nil {
		t.Fatalf("AddAsset: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/service-worker.js", nil)
	response := httptest.NewRecorder()
	registry.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	if got := response.Header().Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("Cache-Control = %q, want no-cache", got)
	}
	if !strings.Contains(response.Body.String(), `self.version = "test"`) {
		t.Fatalf("body = %q, want generated script", response.Body.String())
	}

	manifest := registry.Manifest()
	if len(manifest.Workers) != 2 {
		t.Fatalf("workers = %#v, want 2", manifest.Workers)
	}
	if manifest.Workers[0].Name != "pwa" || manifest.Workers[0].Scope != "/docs/" || !manifest.Workers[0].PWA {
		t.Fatalf("first worker = %#v, want PWA service worker", manifest.Workers[0])
	}

	helpers := registry.Helpers()
	path, err := helpers["worker_path"].(func(string) (string, error))("search")
	if err != nil {
		t.Fatalf("worker_path: %v", err)
	}
	if path != "/assets/search-worker.js" {
		t.Fatalf("worker_path = %q, want /assets/search-worker.js", path)
	}
	fragment, err := helpers["service_worker_register"].(func(string) (lazyview.Fragment, error))("pwa")
	if err != nil {
		t.Fatalf("service_worker_register: %v", err)
	}
	if !strings.Contains(fragment.Body, `navigator.serviceWorker.register("/service-worker.js"`) {
		t.Fatalf("service worker helper = %s", fragment.Body)
	}
}

func TestRegistryRejectsDuplicateNamesAndPaths(t *testing.T) {
	registry := New()
	if err := registry.AddAsset("search", WebWorker, "/assets/search-worker.js"); err != nil {
		t.Fatalf("AddAsset: %v", err)
	}
	if err := registry.AddAsset("search", WebWorker, "/assets/other.js"); err == nil {
		t.Fatal("duplicate name error = nil")
	}
	if err := registry.AddAsset("other", WebWorker, "/assets/search-worker.js"); err == nil {
		t.Fatal("duplicate path error = nil")
	}
}
