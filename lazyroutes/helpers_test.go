package lazyroutes

import (
	"context"
	"net/http"
	"testing"

	"golazy.dev/lazypath"
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
