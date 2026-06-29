//go:build lazydev

package lazyassets

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"golazy.dev/lazycontrolplane"
)

func TestLazyDevAssetsHandlerReturnsRegistryManifest(t *testing.T) {
	registry := New()
	if err := registry.Add("/styles.css", []byte("body{}"), AssetSource("public")); err != nil {
		t.Fatal(err)
	}
	plane := lazycontrolplane.New(lazycontrolplane.Config{})
	RegisterLazyDevHandlers(plane, registry)

	response := httptest.NewRecorder()
	plane.ServeHTTP(response, httptest.NewRequest(http.MethodGet, LazyDevAssetsPath, nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if got := response.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
	var manifest Manifest
	if err := json.Unmarshal(response.Body.Bytes(), &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	if len(manifest.Assets) != 1 || manifest.Assets[0].Path != "/styles.css" || manifest.Assets[0].Source != "public" {
		t.Fatalf("manifest = %#v, want /styles.css public", manifest)
	}
}
