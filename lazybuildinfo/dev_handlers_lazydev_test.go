//go:build lazydev

package lazybuildinfo

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"golazy.dev/lazycontrolplane"
)

func TestRegisterLazyDevHandlersServesBuildInfo(t *testing.T) {
	controlPlane := lazycontrolplane.New(lazycontrolplane.Config{})
	RegisterLazyDevHandlers(controlPlane)

	response := httptest.NewRecorder()
	controlPlane.ServeHTTP(response, httptest.NewRequest(http.MethodGet, LazyDevBuildInfoPath, nil))

	if response.Code != http.StatusOK {
		t.Fatalf("buildinfo status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if got := response.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("buildinfo Content-Type = %q, want JSON", got)
	}
	if got := response.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("buildinfo Cache-Control = %q, want no-store", got)
	}
	var got struct {
		Available bool   `json:"available"`
		GoVersion string `json:"go_version"`
		Path      string `json:"path"`
		Main      struct {
			Path string `json:"path"`
		} `json:"main"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode buildinfo response: %v\n%s", err, response.Body.String())
	}
	if !got.Available {
		t.Fatal("buildinfo available = false, want true")
	}
	if got.GoVersion == "" {
		t.Fatalf("buildinfo go_version is empty: %#v", got)
	}
	if got.Path == "" && got.Main.Path == "" {
		t.Fatalf("buildinfo path and main.path are empty: %#v", got)
	}
}
