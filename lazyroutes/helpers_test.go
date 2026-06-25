package lazyroutes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"golazy.dev/lazypath"
	"golazy.dev/lazyview"
	_ "golazy.dev/lazyview/gotmpl"
)

func TestPathForBuildsNamedRoutePaths(t *testing.T) {
	scope := New(context.Background())
	scope.HandleFunc("GET", "/", func(http.ResponseWriter, *http.Request) error { return nil })
	scope.HandleFunc("GET", "/posts/{post_id}", func(http.ResponseWriter, *http.Request) error { return nil })

	root, err := scope.PathFor("root")
	if err != nil {
		t.Fatal(err)
	}
	if root != "/" {
		t.Fatalf("root path = %q, want /", root)
	}

	post, err := scope.PathFor("posts", "hello world")
	if err != nil {
		t.Fatal(err)
	}
	if post != "/posts/hello%20world" {
		t.Fatalf("post path = %q, want /posts/hello%%20world", post)
	}

	admin, err := scope.PathFor("posts", "hello world", lazypath.URLParams{"token": "secret token"})
	if err != nil {
		t.Fatal(err)
	}
	if admin != "/posts/hello%20world?token=secret+token" {
		t.Fatalf("admin path = %q, want query params", admin)
	}
}

func TestLinkToRendersAnchorWithRouteAndOptions(t *testing.T) {
	scope := New(context.Background())
	scope.HandleFunc("GET", "/posts/{post_id}", func(http.ResponseWriter, *http.Request) error { return nil })

	body, err := renderRouteTemplate(t, scope, "/posts/elsewhere", `{{link_to "A <Post>" (path_for "posts" "hello world") (attr "class" "nav \"primary\"") (data "turbo-method" "post")}}`)
	if err != nil {
		t.Fatal(err)
	}

	want := `<main><a href="/posts/hello%20world" class="nav &#34;primary&#34;" data-turbo-method="post">A &lt;Post&gt;</a></main>`
	if body != want {
		t.Fatalf("rendered body = %q, want %q", body, want)
	}
}

func TestLinkToUnlessCurrentOmitsCurrentPageAnchor(t *testing.T) {
	scope := New(context.Background())
	scope.HandleFunc("GET", "/posts/{post_id}", func(http.ResponseWriter, *http.Request) error { return nil })

	body, err := renderRouteTemplate(t, scope, "/posts/hello%20world?preview=1", `{{link_to "A <Post>" (path_for "posts" "hello world") (unless_current)}}`)
	if err != nil {
		t.Fatal(err)
	}

	want := `<main>A &lt;Post&gt;</main>`
	if body != want {
		t.Fatalf("rendered body = %q, want %q", body, want)
	}
}

func TestLinkDestinationIsCurrentMatchesQueryOnlyWhenDestinationHasQuery(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/posts?page=2", nil)
	if !linkDestinationIsCurrent(request, "/posts") {
		t.Fatal("destination without query should match the current request path")
	}
	if !linkDestinationIsCurrent(request, "/posts?page=2") {
		t.Fatal("destination with matching query should match current request")
	}
	if linkDestinationIsCurrent(request, "/posts?page=3") {
		t.Fatal("destination with different query should not match current request")
	}
	if linkDestinationIsCurrent(request, "https://elsewhere.example/posts?page=2") {
		t.Fatal("destination with different host should not match current request")
	}
}

func TestLinkToRejectsInvalidOptions(t *testing.T) {
	scope := New(context.Background())
	scope.HandleFunc("GET", "/posts", func(http.ResponseWriter, *http.Request) error { return nil })

	_, err := renderRouteTemplate(t, scope, "/posts", `{{link_to "Posts" (path_for "posts") (attr "bad name" "value")}}`)
	if err == nil {
		t.Fatal("render succeeded, want invalid attribute error")
	}
	if !strings.Contains(err.Error(), "invalid character") {
		t.Fatalf("error = %q, want invalid character", err)
	}
}

func TestLinkToRejectsUnsafeHref(t *testing.T) {
	scope := New(context.Background())
	scope.HandleFunc("GET", "/posts", func(http.ResponseWriter, *http.Request) error { return nil })

	_, err := renderRouteTemplate(t, scope, "/posts", `{{link_to "Bad" "javascript:alert(1)"}}`)
	if err == nil {
		t.Fatal("render succeeded, want unsafe href error")
	}
	if !strings.Contains(err.Error(), "scheme") {
		t.Fatalf("error = %q, want scheme error", err)
	}
}

func renderRouteTemplate(t *testing.T, scope *Scope, requestPath string, source string) (string, error) {
	t.Helper()
	views, err := lazyview.New(fstest.MapFS{
		"layouts/app.html.tpl": {Data: []byte(`<main>{{.content}}</main>`)},
		"posts/index.html.tpl": {Data: []byte(source)},
	})
	if err != nil {
		t.Fatal(err)
	}
	views.AddHelpers(scope.RegisterHelpers())
	if err := views.Cache(); err != nil {
		t.Fatal(err)
	}

	var out strings.Builder
	err = views.Render(lazyview.Options{
		Request:    httptest.NewRequest(http.MethodGet, requestPath, nil),
		Writer:     &out,
		Controller: "posts",
		Action:     "index",
		UseLayout:  true,
	})
	return out.String(), err
}
