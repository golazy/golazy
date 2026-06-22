package lazyassets

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"

	"golazy.dev/lazyview"
)

func TestRegistryServesFSAssets(t *testing.T) {
	registry := newBasicRegistry(t)

	response := fetchAsset(registry, http.MethodGet, "/styles.css", nil)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if !strings.Contains(response.Body.String(), "color: black") {
		t.Fatalf("body = %q, want stylesheet", response.Body.String())
	}
	if got := response.Header().Get("Content-Type"); got != "text/css; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want text/css; charset=utf-8", got)
	}
	if got := response.Header().Get("Cache-Control"); got != "public, max-age=0, must-revalidate" {
		t.Fatalf("Cache-Control = %q, want logical cache policy", got)
	}
	if response.Header().Get("ETag") == "" {
		t.Fatal("ETag is empty")
	}
}

func TestRegistryServesHeadWithoutBody(t *testing.T) {
	registry := newBasicRegistry(t)

	response := fetchAsset(registry, http.MethodHead, "/styles.css", nil)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if response.Body.String() != "" {
		t.Fatalf("body = %q, want empty", response.Body.String())
	}
	if response.Header().Get("Content-Length") == "" {
		t.Fatal("Content-Length is empty")
	}
	if response.Header().Get("ETag") == "" {
		t.Fatal("ETag is empty")
	}
}

func TestRegistryUsesPermanentURLs(t *testing.T) {
	registry := newBasicRegistry(t)

	permanent, err := registry.Path("/styles.css")
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^/styles-[a-f0-9]{12}\.css$`).MatchString(permanent) {
		t.Fatalf("permanent path = %q, want hashed stylesheet path", permanent)
	}

	response := fetchAsset(registry, http.MethodGet, permanent, nil)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if !strings.Contains(response.Body.String(), "color: black") {
		t.Fatalf("body = %q, want stylesheet", response.Body.String())
	}
	if got := response.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Fatalf("Cache-Control = %q, want permanent cache policy", got)
	}
}

func TestRegistryStylesheetHelper(t *testing.T) {
	registry := newBasicRegistry(t)
	helper := registry.Helpers()["stylesheet"].(func(string) (lazyview.Fragment, error))

	fragment, err := helper("/styles.css")
	if err != nil {
		t.Fatal(err)
	}
	if fragment.ContentType != "text/html; charset=utf-8" {
		t.Fatalf("ContentType = %q, want text/html; charset=utf-8", fragment.ContentType)
	}
	if !regexp.MustCompile(`^<link rel="stylesheet" href="/styles-[a-f0-9]{12}\.css">$`).MatchString(fragment.Body) {
		t.Fatalf("Body = %q, want stylesheet link with permanent URL", fragment.Body)
	}
}

func TestRegistryAssetBaseURL(t *testing.T) {
	registry := New(WithBaseURL("http://127.0.0.1:8888/buckets/assets/"))
	if err := registry.Add("/styles.css", []byte("body { background: url('/logo.txt') }"), ContentType("text/css")); err != nil {
		t.Fatal(err)
	}
	if err := registry.Add("/logo.txt", []byte("logo")); err != nil {
		t.Fatal(err)
	}

	permanent, err := registry.Path("/styles.css")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(permanent, "http://127.0.0.1:8888/buckets/assets/styles-") {
		t.Fatalf("Path = %q, want base URL", permanent)
	}
	data, err := registry.content("/styles.css")
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`url\('http://127\.0\.0\.1:8888/buckets/assets/logo-[a-f0-9]{12}\.txt'\)`).Match(data) {
		t.Fatalf("stylesheet = %q, want rewritten absolute asset URL", data)
	}
}

func TestRegistryImportmapHelper(t *testing.T) {
	registry := New()
	if err := registry.Add("/assets/importmap.json", []byte(`{"imports":{"library":"/assets/library.js","unsafe":"</script>"}}`)); err != nil {
		t.Fatal(err)
	}

	helper := registry.Helpers()["importmap"].(func(string) (lazyview.Fragment, error))
	fragment, err := helper("/assets/importmap.json")
	if err != nil {
		t.Fatal(err)
	}
	if fragment.ContentType != "text/html; charset=utf-8" {
		t.Fatalf("ContentType = %q, want text/html; charset=utf-8", fragment.ContentType)
	}
	if !strings.HasPrefix(fragment.Body, `<script type="importmap">`) {
		t.Fatalf("Body = %q, want importmap script", fragment.Body)
	}
	if strings.Contains(fragment.Body, "</script>") && !strings.HasSuffix(fragment.Body, "</script>") {
		t.Fatalf("Body contains an unescaped closing script tag: %q", fragment.Body)
	}
	if !strings.Contains(fragment.Body, `\u003c/script\u003e`) {
		t.Fatalf("Body = %q, want escaped JSON", fragment.Body)
	}
}

func TestRegistryImportmapHelperRewritesAssetBaseURL(t *testing.T) {
	registry := New(WithBaseURL("http://127.0.0.1:8888/buckets/assets"))
	if err := registry.Add("/assets/importmap.json", []byte(`{"imports":{"library":"/assets/library.js","remote":"https://cdn.example.test/x.js"}}`)); err != nil {
		t.Fatal(err)
	}

	helper := registry.Helpers()["importmap"].(func(string) (lazyview.Fragment, error))
	fragment, err := helper("/assets/importmap.json")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(fragment.Body, `http://127.0.0.1:8888/buckets/assets/assets/library.js`) {
		t.Fatalf("Body = %q, want local import rewritten to asset base URL", fragment.Body)
	}
	if !strings.Contains(fragment.Body, `https://cdn.example.test/x.js`) {
		t.Fatalf("Body = %q, want remote import preserved", fragment.Body)
	}
}

func TestRegistryHandlesIfNoneMatch(t *testing.T) {
	registry := newBasicRegistry(t)
	first := fetchAsset(registry, http.MethodGet, "/styles.css", nil)
	etag := first.Header().Get("ETag")
	if etag == "" {
		t.Fatal("ETag is empty")
	}

	response := fetchAsset(registry, http.MethodGet, "/styles.css", func(req *http.Request) {
		req.Header.Set("If-None-Match", etag)
	})
	if response.Code != http.StatusNotModified {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotModified)
	}
	if response.Body.String() != "" {
		t.Fatalf("body = %q, want empty", response.Body.String())
	}
	if response.Header().Get("ETag") != etag {
		t.Fatalf("ETag = %q, want %q", response.Header().Get("ETag"), etag)
	}
	if response.Header().Get("Content-Type") != "" {
		t.Fatalf("Content-Type = %q, want empty", response.Header().Get("Content-Type"))
	}
}

func TestRegistryRewritesCSSURLsToPermanentAssetPaths(t *testing.T) {
	registry := New()
	if err := registry.AddFS(os.DirFS("testdata/cases/css-url-rewrite/public")); err != nil {
		t.Fatal(err)
	}

	response := fetchAsset(registry, http.MethodGet, "/css/app.css", nil)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	body := response.Body.String()
	if !regexp.MustCompile(`/images/logo-[a-f0-9]{12}\.txt`).MatchString(body) {
		t.Fatalf("body = %q, want rewritten relative logo URL", body)
	}
	if !regexp.MustCompile(`/images/icon-[a-f0-9]{12}\.txt#shape`).MatchString(body) {
		t.Fatalf("body = %q, want rewritten root icon URL with fragment", body)
	}
	if !strings.Contains(body, "data:image/png;base64,abc") {
		t.Fatalf("body = %q, want data URL preserved", body)
	}
	if !strings.Contains(body, "https://example.test/remote.png") {
		t.Fatalf("body = %q, want remote URL preserved", body)
	}
}

func TestRegistryCanDisableCSSURLRewrite(t *testing.T) {
	registry := New(WithCSSURLRewrite(false))
	if err := registry.AddFS(os.DirFS("testdata/cases/css-url-rewrite/public")); err != nil {
		t.Fatal(err)
	}

	response := fetchAsset(registry, http.MethodGet, "/css/app.css", nil)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if !strings.Contains(response.Body.String(), `url("../images/logo.txt")`) {
		t.Fatalf("body = %q, want original CSS URL", response.Body.String())
	}
}

func TestRegistrySupportsGeneratedAssets(t *testing.T) {
	registry := New()
	if err := registry.Add("/generated/app.js", []byte("console.log('generated');"), ContentType("text/javascript")); err != nil {
		t.Fatal(err)
	}

	permanent, err := registry.Path("/generated/app.js")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(permanent, "/generated/app-") {
		t.Fatalf("permanent path = %q, want generated hashed path", permanent)
	}

	response := fetchAsset(registry, http.MethodGet, permanent, nil)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if response.Body.String() != "console.log('generated');" {
		t.Fatalf("body = %q, want generated JavaScript", response.Body.String())
	}

	manifest := registry.Manifest()
	if len(manifest.Assets) != 1 {
		t.Fatalf("manifest assets = %d, want 1", len(manifest.Assets))
	}
	if !manifest.Assets[0].Generated {
		t.Fatal("manifest asset Generated = false, want true")
	}
	if manifest.Assets[0].Integrity == "" {
		t.Fatal("manifest asset Integrity is empty")
	}
}

func TestRegistryRejectsDuplicateAssetsUnlessReplaced(t *testing.T) {
	registry := New()
	if err := registry.Add("/app.js", []byte("first")); err != nil {
		t.Fatal(err)
	}
	if err := registry.Add("/app.js", []byte("second")); err == nil {
		t.Fatal("duplicate Add error = nil, want error")
	}
	if err := registry.Add("/app.js", []byte("second"), ReplaceAsset()); err != nil {
		t.Fatal(err)
	}

	response := fetchAsset(registry, http.MethodGet, "/app.js", nil)
	if response.Body.String() != "second" {
		t.Fatalf("body = %q, want replacement content", response.Body.String())
	}
}

func TestRegistrySupportsURLPrefix(t *testing.T) {
	registry := New(WithURLPrefix("/assets"))
	if err := registry.Add("/app.js", []byte("app")); err != nil {
		t.Fatal(err)
	}

	permanent, err := registry.Path("/app.js")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(permanent, "/assets/app-") {
		t.Fatalf("permanent = %q, want /assets/app-...", permanent)
	}

	response := fetchAsset(registry, http.MethodGet, "/app.js", nil)
	if response.Code != http.StatusNotFound {
		t.Fatalf("unprefixed status = %d, want %d", response.Code, http.StatusNotFound)
	}
	response = fetchAsset(registry, http.MethodGet, "/assets/app.js", nil)
	if response.Code != http.StatusOK {
		t.Fatalf("prefixed status = %d, want %d", response.Code, http.StatusOK)
	}
}

func TestRegistryFallsThroughAndRejectsUnsupportedMethods(t *testing.T) {
	registry := newBasicRegistry(t)
	handler := registry.Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("next"))
	}))

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/missing.txt", nil))
	if response.Code != http.StatusCreated {
		t.Fatalf("missing status = %d, want %d", response.Code, http.StatusCreated)
	}
	if response.Body.String() != "next" {
		t.Fatalf("missing body = %q, want next", response.Body.String())
	}

	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/styles.css", nil))
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("post status = %d, want %d", response.Code, http.StatusMethodNotAllowed)
	}
	if response.Header().Get("Allow") != "GET, HEAD" {
		t.Fatalf("Allow = %q, want GET, HEAD", response.Header().Get("Allow"))
	}
}

func TestRegistryServesIndexForRoot(t *testing.T) {
	registry := newBasicRegistry(t)

	response := fetchAsset(registry, http.MethodGet, "/", nil)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if response.Body.String() != "hello index\n" {
		t.Fatalf("body = %q, want index", response.Body.String())
	}
}

func TestRegistryIgnoresOversizedDiskAssetsForPipeline(t *testing.T) {
	registry := New(WithMaxAssetSize(4))
	err := registry.AddFS(fstest.MapFS{
		"large.txt": {Data: []byte("12345")},
	})
	if err != nil {
		t.Fatal(err)
	}

	assetPath, err := registry.Path("/large.txt")
	if err != nil {
		t.Fatal(err)
	}
	if assetPath != "/large.txt" {
		t.Fatalf("asset path = %q, want logical path", assetPath)
	}

	response := fetchAsset(registry, http.MethodGet, "/large.txt", nil)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if response.Body.String() != "12345" {
		t.Fatalf("body = %q, want oversized content", response.Body.String())
	}
	if response.Header().Get("ETag") != "" {
		t.Fatalf("ETag = %q, want empty", response.Header().Get("ETag"))
	}

	manifest := registry.Manifest()
	if len(manifest.Assets) != 1 || !manifest.Assets[0].Ignored {
		t.Fatalf("manifest = %#v, want one ignored asset", manifest.Assets)
	}
}

func TestRegistryRejectsOversizedGeneratedAssetsByDefault(t *testing.T) {
	registry := New(WithMaxAssetSize(4))
	if err := registry.Add("/large.txt", []byte("12345")); err == nil {
		t.Fatal("Add oversized generated asset error = nil, want error")
	}
}

func TestRegistryUnpacksLogicalPermanentAndManifestFiles(t *testing.T) {
	registry := newBasicRegistry(t)
	dir := t.TempDir()
	if err := registry.Unpack(dir); err != nil {
		t.Fatal(err)
	}

	logical, err := os.ReadFile(filepath.Join(dir, "styles.css"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(logical), "color: black") {
		t.Fatalf("logical file = %q, want stylesheet", string(logical))
	}

	permanent, err := registry.Path("/styles.css")
	if err != nil {
		t.Fatal(err)
	}
	permanentData, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(strings.TrimPrefix(permanent, "/"))))
	if err != nil {
		t.Fatal(err)
	}
	if string(permanentData) != string(logical) {
		t.Fatalf("permanent file = %q, want %q", string(permanentData), string(logical))
	}

	manifest, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(manifest), `"permanent"`) {
		t.Fatalf("manifest = %s, want permanent entries", manifest)
	}
}

func TestRegistryRejectsEscapingPaths(t *testing.T) {
	registry := New()
	if err := registry.Add("../secret.txt", []byte("no")); err == nil {
		t.Fatal("Add escaping path error = nil, want error")
	}
}

func newBasicRegistry(t *testing.T) *Registry {
	t.Helper()
	registry := New()
	if err := registry.AddFS(os.DirFS("testdata/cases/basic/public")); err != nil {
		t.Fatal(err)
	}
	return registry
}

func fetchAsset(registry *Registry, method, target string, configure func(*http.Request)) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	request := httptest.NewRequest(method, target, nil)
	if configure != nil {
		configure(request)
	}
	registry.ServeHTTP(response, request)
	return response
}
