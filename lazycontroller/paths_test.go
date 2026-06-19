package lazycontroller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"golazy.dev/lazyview"
	_ "golazy.dev/lazyview/gotmpl"
)

func TestPathForUsesConfiguredHelper(t *testing.T) {
	base, response := newPathForTestBase(t, true)
	request := httptest.NewRequest(http.MethodGet, "https://example.com/posts", nil)
	if err := base.BindRequest(response, request, lazyview.Route{Controller: "posts"}); err != nil {
		t.Fatal(err)
	}

	path, err := base.PathFor("post", "hello world")
	if err != nil {
		t.Fatal(err)
	}

	if got, want := path, "/post/hello world"; got != want {
		t.Fatalf("PathFor = %q, want %q", got, want)
	}
}

func TestPathForErrorsWithoutConfiguredHelper(t *testing.T) {
	base, response := newPathForTestBase(t, false)
	request := httptest.NewRequest(http.MethodGet, "https://example.com/posts", nil)
	if err := base.BindRequest(response, request, lazyview.Route{Controller: "posts"}); err != nil {
		t.Fatal(err)
	}

	if _, err := base.PathFor("post", "hello"); err == nil {
		t.Fatal("expected missing path helper error")
	}
}

func newPathForTestBase(t *testing.T, withPathFor bool) (Base, *httptest.ResponseRecorder) {
	t.Helper()
	renderer, err := NewRenderer(fstest.MapFS{
		"layouts/app.html.tpl": {Data: []byte(`{{.content}}`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := WithRenderer(context.Background(), renderer)
	if withPathFor {
		ctx = WithPathFor(ctx, func(name string, values ...any) (string, error) {
			path := "/" + name
			for _, value := range values {
				path += "/" + value.(string)
			}
			return path, nil
		})
	}
	base, err := NewBase(ctx)
	if err != nil {
		t.Fatal(err)
	}
	return base, httptest.NewRecorder()
}
