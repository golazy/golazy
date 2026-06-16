package lazycontroller

import (
	"context"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"golazy.dev/lazyview"
	_ "golazy.dev/lazyview/gotmpl"
)

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
