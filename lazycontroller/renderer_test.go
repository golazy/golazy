package lazycontroller

import (
	"context"
	"io/fs"
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
	ctx = WithWriter(ctx, response)
	ctx = WithRoute(ctx, lazyview.Route{Controller: "posts"})
	base, err := NewBase(ctx)
	if err != nil {
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
	ctx = WithWriter(ctx, httptest.NewRecorder())
	ctx = WithRoute(ctx, lazyview.Route{Controller: "posts"})
	base, err := NewBase(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := base.Render("missing"); err == nil {
		t.Fatal("expected missing view error")
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
