package lazycontroller

import (
	"context"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"golazy.dev/lazycache"
	"golazy.dev/lazyseo"
	"golazy.dev/lazyseo/jsonld"
	"golazy.dev/lazyview"
	_ "golazy.dev/lazyview/gotmpl"
)

type rendererCacheBackend struct {
	values map[string]any
	keys   []string
}

type rendererAppContextKey struct{}
type rendererRequestContextKey struct{}

func (b *rendererCacheBackend) Get(key string) (any, error) {
	if value, ok := b.values[key]; ok {
		return value, nil
	}
	return nil, lazycache.ErrMiss
}

func (b *rendererCacheBackend) Set(key string, value any) error {
	if b.values == nil {
		b.values = map[string]any{}
	}
	b.values[key] = value
	b.keys = append(b.keys, key)
	return nil
}

func (b *rendererCacheBackend) Stats() lazycache.Stats {
	return lazycache.Stats{Entries: len(b.values)}
}

func TestBindRequestUsesInheritedAppContextWithoutCloningRequest(t *testing.T) {
	renderer, err := NewRenderer(fstest.MapFS{
		"layouts/app.html.tpl": {Data: []byte(`{{.content}}`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	appCtx := context.WithValue(WithRenderer(context.Background(), renderer), rendererAppContextKey{}, "app")
	requestCtx := context.WithValue(appCtx, rendererRequestContextKey{}, "request")
	request := httptest.NewRequest(http.MethodGet, "/posts", nil).WithContext(requestCtx)
	base, err := NewBase(appCtx)
	if err != nil {
		t.Fatal(err)
	}

	if err := base.BindRequest(httptest.NewRecorder(), request, lazyview.Route{Controller: "posts"}); err != nil {
		t.Fatal(err)
	}

	if base.Request() != request {
		t.Fatal("BindRequest cloned a request that already inherited the app context")
	}
	if got := base.Request().Context().Value(rendererAppContextKey{}); got != "app" {
		t.Fatalf("app context value = %v, want app", got)
	}
	if got := base.Request().Context().Value(rendererRequestContextKey{}); got != "request" {
		t.Fatalf("request context value = %v, want request", got)
	}
}

func TestBindRequestMergesAppContextWhenRequestDidNotInheritIt(t *testing.T) {
	renderer, err := NewRenderer(fstest.MapFS{
		"layouts/app.html.tpl": {Data: []byte(`{{.content}}`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	appCtx := context.WithValue(WithRenderer(context.Background(), renderer), rendererAppContextKey{}, "app")
	requestCtx := context.WithValue(context.Background(), rendererRequestContextKey{}, "request")
	request := httptest.NewRequest(http.MethodGet, "/posts", nil).WithContext(requestCtx)
	base, err := NewBase(appCtx)
	if err != nil {
		t.Fatal(err)
	}

	if err := base.BindRequest(httptest.NewRecorder(), request, lazyview.Route{Controller: "posts"}); err != nil {
		t.Fatal(err)
	}

	if base.Request() == request {
		t.Fatal("BindRequest reused a request that did not inherit the app context")
	}
	if got := base.Request().Context().Value(rendererAppContextKey{}); got != "app" {
		t.Fatalf("app context value = %v, want app", got)
	}
	if got := base.Request().Context().Value(rendererRequestContextKey{}); got != "request" {
		t.Fatalf("request context value = %v, want request", got)
	}
}

func TestPooledBufferedResponseClearsState(t *testing.T) {
	first := newBufferedResponse()
	first.Header().Set("X-Test", "stale")
	first.WriteHeader(http.StatusCreated)
	_, _ = first.Write([]byte("stale"))
	releaseBufferedResponse(first)

	second := newBufferedResponse()
	defer releaseBufferedResponse(second)
	if got := second.Header().Get("X-Test"); got != "" {
		t.Fatalf("pooled header = %q, want empty", got)
	}
	if second.sent {
		t.Fatal("pooled response was marked sent")
	}
	if got := second.body.Len(); got != 0 {
		t.Fatalf("pooled body length = %d, want 0", got)
	}
}

func TestRenderEscapesDataAndComposesLayout(t *testing.T) {
	views := fstest.MapFS{
		"layouts/app.html.tpl": {Data: []byte(`{{$content := .content}}<main>{{$content}}</main>`)},
		"posts/index.html.tpl": {Data: []byte(`<p>{{.value}}</p>`)},
	}
	renderer, err := NewRenderer(views)
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	ctx := WithRenderer(context.Background(), renderer)
	base, err := NewBase(ctx)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/posts", nil)
	if err := base.BindRequest(response, request, lazyview.Route{Controller: "posts"}); err != nil {
		t.Fatal(err)
	}
	base.Set("value", `<script>unsafe()</script>`)
	if err := base.Render("index"); err != nil {
		t.Fatal(err)
	}

	body := response.Body.String()
	if !strings.Contains(body, `<main><p>&lt;script&gt;unsafe()&lt;/script&gt;</p></main>`) {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestRenderUsesSEOControllerHelpers(t *testing.T) {
	views := fstest.MapFS{
		"layouts/app.html.tpl": {Data: []byte(`<html lang="{{seo_lang}}"><head>{{seo}}</head><main>{{.content}}</main></html>`)},
		"posts/show.html.tpl":  {Data: []byte(`post`)},
	}
	renderer, err := NewRenderer(views)
	if err != nil {
		t.Fatal(err)
	}
	renderer.AddHelpers(lazyseo.Helpers(lazyseo.SiteName("GoLazy")))
	response := httptest.NewRecorder()
	ctx := WithRenderer(context.Background(), renderer)
	base, err := NewBase(ctx)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/posts/hello", nil)
	if err := base.BindRequest(response, request, lazyview.Route{Controller: "posts"}); err != nil {
		t.Fatal(err)
	}

	base.Title("Hello")
	base.Description("A post")
	base.Language("en")
	base.Canonical("https://golazy.dev/posts/hello")
	base.Alternate("de", "https://golazy.dev/de/posts/hello")
	base.SEOImage("https://golazy.dev/posts/hello.png")
	base.SEOImageAlt("Hello post preview")
	base.Kind(lazyseo.Article)
	base.JSONLD(jsonld.NewArticle("Hello"))

	if err := base.Render("show"); err != nil {
		t.Fatal(err)
	}

	body := response.Body.String()
	for _, expected := range []string{
		`<html lang="en">`,
		`<title>Hello - GoLazy</title>`,
		`<meta name="description" content="A post">`,
		`<link rel="canonical" href="https://golazy.dev/posts/hello">`,
		`<link rel="alternate" hreflang="de" href="https://golazy.dev/de/posts/hello">`,
		`<meta property="og:type" content="article">`,
		`<meta property="og:image" content="https://golazy.dev/posts/hello.png">`,
		`<meta property="og:image:alt" content="Hello post preview">`,
		`<meta name="twitter:image:alt" content="Hello post preview">`,
		`<script type="application/ld+json">{"@context":"https://schema.org","@type":"Article","headline":"Hello"}</script>`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("body does not contain %q:\n%s", expected, body)
		}
	}
}

type metadataPost struct{}

func (metadataPost) Title() string {
	return "Metadata Post"
}

func (metadataPost) Description() string {
	return "Base description"
}

func (metadataPost) Canonical() string {
	return "https://golazy.dev/posts/metadata"
}

func (metadataPost) Image() string {
	return "https://golazy.dev/posts/metadata.png"
}

func (metadataPost) ImageAlt() string {
	return "Metadata post preview"
}

func (metadataPost) PublishedTime() time.Time {
	return time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
}

func (metadataPost) Kind() lazyseo.PageKind {
	return lazyseo.Article
}

func (metadataPost) LastUpdated() time.Time {
	return time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
}

func (metadataPost) OpenGraph() lazyseo.OpenGraph {
	return lazyseo.OpenGraph{
		Description: "Open Graph description",
		Image:       "https://golazy.dev/posts/metadata-og.png",
	}
}

func (metadataPost) TwitterCard() lazyseo.TwitterCard {
	return lazyseo.TwitterCard{
		Card:        "summary_large_image",
		Description: "Twitter description",
		Image:       "https://golazy.dev/posts/metadata-twitter.png",
	}
}

func TestRenderUsesMetadataModelInterfaces(t *testing.T) {
	views := fstest.MapFS{
		"layouts/app.html.tpl": {Data: []byte(`<head>{{seo}}</head><main>{{.content}}</main>`)},
		"posts/show.html.tpl":  {Data: []byte(`post`)},
	}
	renderer, err := NewRenderer(views)
	if err != nil {
		t.Fatal(err)
	}
	renderer.AddHelpers(lazyseo.Helpers(lazyseo.SiteName("GoLazy")))
	response := httptest.NewRecorder()
	ctx := WithRenderer(context.Background(), renderer)
	base, err := NewBase(ctx)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/posts/metadata", nil)
	if err := base.BindRequest(response, request, lazyview.Route{Controller: "posts"}); err != nil {
		t.Fatal(err)
	}

	base.Metadata(metadataPost{})
	base.Alternate("de", "https://golazy.dev/de/posts/metadata")

	if err := base.Render("show"); err != nil {
		t.Fatal(err)
	}

	body := response.Body.String()
	for _, expected := range []string{
		`<title>Metadata Post - GoLazy</title>`,
		`<meta name="description" content="Base description">`,
		`<meta property="og:description" content="Open Graph description">`,
		`<meta property="og:image" content="https://golazy.dev/posts/metadata-og.png">`,
		`<meta property="og:image:alt" content="Metadata post preview">`,
		`<meta name="twitter:card" content="summary_large_image">`,
		`<meta name="twitter:description" content="Twitter description">`,
		`<meta name="twitter:image" content="https://golazy.dev/posts/metadata-twitter.png">`,
		`<meta name="twitter:image:alt" content="Metadata post preview">`,
		`<meta property="article:published_time" content="2026-06-19T12:00:00Z">`,
		`<link rel="alternate" hreflang="de" href="https://golazy.dev/de/posts/metadata">`,
		`<script type="application/ld+json">{"@context":"https://schema.org","@type":"Article","headline":"Metadata Post","description":"Base description","url":"https://golazy.dev/posts/metadata","image":"https://golazy.dev/posts/metadata.png","dateModified":"2026-06-20"}</script>`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("body does not contain %q:\n%s", expected, body)
		}
	}
}

func TestRenderSVGStringUsesVariants(t *testing.T) {
	views := fstest.MapFS{
		"layouts/app.html.tpl":       {Data: []byte(`{{.content}}`)},
		"posts/show.svg+square.tpl":  {Data: []byte(`<svg>{{.title}} square</svg>`)},
		"posts/show.svg.tpl":         {Data: []byte(`<svg>{{.title}}</svg>`)},
		"posts/preview.svg+wide.tpl": {Data: []byte(`<svg>{{.title}} wide</svg>`)},
		"posts/preview.svg.tpl":      {Data: []byte(`<svg>{{.title}} preview</svg>`)},
	}
	renderer, err := NewRenderer(views)
	if err != nil {
		t.Fatal(err)
	}
	ctx := WithRenderer(context.Background(), renderer)
	base, err := NewBase(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := base.BindRequest(
		httptest.NewRecorder(),
		httptest.NewRequest(http.MethodGet, "/posts/hello", nil),
		lazyview.Route{Controller: "posts", Action: "Show"},
	); err != nil {
		t.Fatal(err)
	}
	base.Set("title", "Hello")

	body, err := base.RenderSVGString("", "square")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := body, `<svg>Hello square</svg>`; got != want {
		t.Fatalf("svg = %q, want %q", got, want)
	}

	body, err = base.RenderSVGString("preview", "wide")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := body, `<svg>Hello wide</svg>`; got != want {
		t.Fatalf("svg = %q, want %q", got, want)
	}
}

func TestRenderMissingView(t *testing.T) {
	views := fstest.MapFS{
		"layouts/app.html.tpl": {Data: []byte(`{{.content}}`)},
	}
	renderer, err := NewRenderer(views)
	if err != nil {
		t.Fatal(err)
	}
	ctx := WithRenderer(context.Background(), renderer)
	base, err := NewBase(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := base.BindRequest(
		httptest.NewRecorder(),
		httptest.NewRequest(http.MethodGet, "/posts", nil),
		lazyview.Route{Controller: "posts"},
	); err != nil {
		t.Fatal(err)
	}
	if err := base.Render("missing"); err == nil {
		t.Fatal("expected missing view error")
	}
}

func TestRenderUsesResponseHelpers(t *testing.T) {
	views := fstest.MapFS{
		"layouts/app.html.tpl": {Data: []byte(`layout {{.content}}`)},
		"posts/create.html.tpl": {
			Data: []byte(`created`),
		},
	}
	renderer, err := NewRenderer(views)
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	ctx := WithRenderer(context.Background(), renderer)
	base, err := NewBase(ctx)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/posts", nil)
	if err := base.BindRequest(response, request, lazyview.Route{Controller: "posts"}); err != nil {
		t.Fatal(err)
	}
	base.Status(http.StatusCreated)
	base.Header().Set("Cache-Control", "no-store")
	base.ContentType("text/plain; charset=utf-8")

	if err := base.Render("create"); err != nil {
		t.Fatal(err)
	}

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusCreated)
	}
	if got, want := response.Header().Get("Cache-Control"), "no-store"; got != want {
		t.Fatalf("Cache-Control = %q, want %q", got, want)
	}
	if got, want := response.Header().Get("Content-Type"), "text/plain; charset=utf-8"; got != want {
		t.Fatalf("Content-Type = %q, want %q", got, want)
	}
	if got, want := response.Body.String(), "layout created"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestRenderUsesControllerCacheKey(t *testing.T) {
	views := fstest.MapFS{
		"layouts/app.html.tpl":      {Data: []byte(`<main>{{.content}}</main>`)},
		"admin/posts/show.html.tpl": {Data: []byte(`<p>{{.title}}</p>`)},
		"posts/show.html.tpl":       {Data: []byte(`wrong`)},
	}
	renderer, err := NewRenderer(views)
	if err != nil {
		t.Fatal(err)
	}
	backend := &rendererCacheBackend{}
	cache, err := lazycache.New(lazycache.Options{Backend: backend})
	if err != nil {
		t.Fatal(err)
	}
	ctx := lazycache.WithBuildVersion(lazycache.WithCache(WithRenderer(context.Background(), renderer), cache), "devel")

	first := httptest.NewRecorder()
	base, err := NewBase(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := base.BindRequest(first, httptest.NewRequest(http.MethodGet, "/admin/posts/1", nil), lazyview.Route{
		Namespace:  "admin",
		Controller: "posts",
		Action:     "Show",
	}); err != nil {
		t.Fatal(err)
	}
	base.Set("title", "First")
	base.Variants("compact")
	if base.CacheKey(1, "stamp") {
		t.Fatal("CacheKey returned true before cache was populated")
	}
	if err := base.Render(""); err != nil {
		t.Fatal(err)
	}

	second := httptest.NewRecorder()
	base, err = NewBase(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := base.BindRequest(second, httptest.NewRequest(http.MethodGet, "/admin/posts/1", nil), lazyview.Route{
		Namespace:  "admin",
		Controller: "posts",
		Action:     "Show",
	}); err != nil {
		t.Fatal(err)
	}
	base.Set("title", "Second")
	base.Variants("compact")
	if !base.CacheKey(1, "stamp") {
		t.Fatal("CacheKey returned false for populated cache")
	}
	if err := base.Render(""); err != nil {
		t.Fatal(err)
	}

	if got, want := first.Body.String(), `<main><p>First</p></main>`; got != want {
		t.Fatalf("first body = %q, want %q", got, want)
	}
	if got, want := second.Body.String(), first.Body.String(); got != want {
		t.Fatalf("second body = %q, want cached %q", got, want)
	}
	if len(backend.keys) != 1 {
		t.Fatalf("cache writes = %v, want one write", backend.keys)
	}
	if got, want := backend.keys[0], "build-devel-variant-compact-admin-posts-show-html-1-stamp"; got != want {
		t.Fatalf("cache key = %q, want %q", got, want)
	}
}

func TestRenderUsesControllerFullCacheKey(t *testing.T) {
	views := fstest.MapFS{
		"layouts/app.html.tpl": {Data: []byte(`{{.content}}`)},
		"posts/show.html.tpl":  {Data: []byte(`{{.title}}`)},
	}
	renderer, err := NewRenderer(views)
	if err != nil {
		t.Fatal(err)
	}
	backend := &rendererCacheBackend{}
	cache, err := lazycache.New(lazycache.Options{Backend: backend})
	if err != nil {
		t.Fatal(err)
	}
	ctx := lazycache.WithBuildVersion(lazycache.WithCache(WithRenderer(context.Background(), renderer), cache), "v1.2.3")
	response := httptest.NewRecorder()
	base, err := NewBase(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := base.BindRequest(response, httptest.NewRequest(http.MethodGet, "/posts/1", nil), lazyview.Route{
		Controller: "posts",
		Action:     "Show",
	}); err != nil {
		t.Fatal(err)
	}
	base.Set("title", "Post")
	base.Variants("phone")
	if base.CacheKeyF("post", 1, "stamp") {
		t.Fatal("CacheKeyF returned true before cache was populated")
	}
	if err := base.Render(""); err != nil {
		t.Fatal(err)
	}
	if got, want := backend.keys[0], "build-v1.2.3-variant-phone-post-1-stamp"; got != want {
		t.Fatalf("cache key = %q, want %q", got, want)
	}
}

func TestRenderSelectsLayout(t *testing.T) {
	views := fstest.MapFS{
		"layouts/app.html.tpl":   {Data: []byte(`app {{.content}}`)},
		"layouts/admin.html.tpl": {Data: []byte(`admin {{.content}}`)},
		"posts/index.html.tpl":   {Data: []byte(`posts`)},
	}
	renderer, err := NewRenderer(views)
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	ctx := WithRenderer(context.Background(), renderer)
	base, err := NewBase(ctx)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/posts", nil)
	if err := base.BindRequest(response, request, lazyview.Route{Controller: "posts"}); err != nil {
		t.Fatal(err)
	}
	base.Layout("admin")

	if err := base.Render("index"); err != nil {
		t.Fatal(err)
	}

	if got, want := response.Body.String(), "admin posts"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestRenderCanSkipLayout(t *testing.T) {
	views := fstest.MapFS{
		"layouts/app.html.tpl": {Data: []byte(`layout {{.content}}`)},
		"posts/index.html.tpl": {Data: []byte(`posts`)},
	}
	renderer, err := NewRenderer(views)
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	ctx := WithRenderer(context.Background(), renderer)
	base, err := NewBase(ctx)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/posts", nil)
	if err := base.BindRequest(response, request, lazyview.Route{Controller: "posts"}); err != nil {
		t.Fatal(err)
	}
	base.NoLayout()

	if err := base.Render("index"); err != nil {
		t.Fatal(err)
	}

	if got, want := response.Body.String(), "posts"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestBaseExposesRequest(t *testing.T) {
	views := fstest.MapFS{
		"layouts/app.html.tpl": {Data: []byte(`{{.content}}`)},
	}
	renderer, err := NewRenderer(views)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/posts", nil)
	ctx := WithRenderer(context.Background(), renderer)
	base, err := NewBase(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := base.BindRequest(httptest.NewRecorder(), request, lazyview.Route{Controller: "posts"}); err != nil {
		t.Fatal(err)
	}
	if base.Request() == nil {
		t.Fatal("base Request returned nil")
	}
	if base.Request().Method != request.Method || base.Request().URL.Path != request.URL.Path {
		t.Fatalf("base Request = %s %s, want %s %s", base.Request().Method, base.Request().URL.Path, request.Method, request.URL.Path)
	}
}

func TestReturnFileWritesPublicFileWithStatus(t *testing.T) {
	renderer, err := NewRenderer(fstest.MapFS{
		"layouts/app.html.tpl": {Data: []byte(`{{.content}}`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	ctx := WithRenderer(context.Background(), renderer)
	ctx = WithErrorPages(ctx, fstest.MapFS{
		"404.html": {Data: []byte("<h1>missing</h1>")},
	})
	base, err := NewBase(ctx)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/posts/missing", nil)
	if err := base.BindRequest(response, request, lazyview.Route{Controller: "posts"}); err != nil {
		t.Fatal(err)
	}

	if err := base.ReturnFile("404.html", http.StatusNotFound); err != nil {
		t.Fatal(err)
	}

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
	if got, want := response.Body.String(), "<h1>missing</h1>"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
	if got, want := response.Header().Get("Content-Type"), "text/html; charset=utf-8"; got != want {
		t.Fatalf("Content-Type = %q, want %q", got, want)
	}
}

func TestNewRendererRequiresLayout(t *testing.T) {
	_, err := NewRenderer(fstest.MapFS{})
	if err == nil {
		t.Fatal("expected missing layout error")
	}
	if _, ok := err.(*fs.PathError); ok {
		t.Fatalf("expected contextual error, got %v", err)
	}
}
