package lazyapp

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golazy.dev/lazypwa"
)

func TestAppWiresPWAAndWorkers(t *testing.T) {
	app := New(Config{
		Name: "example",
		PWA: lazypwa.Config{
			Installable: true,
			Version:     "v1.2.3",
			Manifest: lazypwa.ManifestConfig{
				Name: "Example",
			},
		},
	})

	if app.PWA == nil || !app.PWA.Enabled() {
		t.Fatal("app PWA is not enabled")
	}
	if _, ok := app.Workers.Worker("pwa"); !ok {
		t.Fatal("PWA service worker is not registered")
	}

	response := httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/manifest.webmanifest", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("manifest status = %d, want 200", response.Code)
	}
	if !strings.Contains(response.Body.String(), `"name": "Example"`) {
		t.Fatalf("manifest body = %s", response.Body.String())
	}

	response = httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/service-worker.js", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("worker status = %d, want 200", response.Code)
	}
	if !strings.Contains(response.Body.String(), `LAZYPWA_VERSION = "v1.2.3"`) {
		t.Fatalf("worker body = %s", response.Body.String())
	}
}
