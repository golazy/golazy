package lazydispatch

import (
	"fmt"
	"testing"
)

func TestRoute_Normalize(t *testing.T) {

	r := Route{
		URL: "posts/:id",
	}
	r.normalize()
	if r.Method != "GET" {
		t.Errorf("expected method to be GET, got %s", r.Method)
	}
	if len(r.Models) != 1 {
		t.Errorf("expected models to have 1 item, got %d", len(r.Models))
	}

}

type User struct {
	ID int
}
type Post struct {
	ID int
}
type Comment struct {
	ID int
}

type args []any

func TestReplaceParams(t *testing.T) {

	out := buildRoute(&Route{Path: "/posts/:id/comments/:id"}, "1", "2")
	if out != "/posts/1/comments/2" {
		t.Fatalf("expected /posts/1/comments/2, got %s", out)
	}
	out = buildRoute(&Route{Path: ":id"}, "1")
	if out != "1" {
		t.Fatalf("expected \"1\", got %s", out)
	}

}

func TestPathFor(t *testing.T) {
	nr := newNamedRoutes()

	// Simple without params
	nr.Add(Route{Path: "/posts", Name: "posts"})
	nr.Add(Route{Path: "/posts/:id", Name: "post", Models: []any{any(struct{}{})}})
	nr.Add(Route{Path: "/comments/:comment_id", Name: "post", Models: []any{&Comment{}}})
	nr.Add(Route{Path: "/posts/:post_id/comments/:comment_id", Name: "post_comment", Models: []any{&Post{}, &Comment{}}})

	for i, r := range nr.routes {
		t.Log(i, r.String())
	}

	expectPath := func(expected string, args ...any) {
		t.Helper()
		func() {
			defer func() {
				err := recover()
				if err != nil {
					t.Errorf("expected %q to not have error, got %s", expected, err)
				}
			}()
			out := nr.PathFor(args...)
			if out != expected {
				t.Errorf("expected %s, got %s", expected, out)
			}
		}()
	}
	expectError := func(eerr string, args ...any) {
		t.Helper()
		func() {
			defer func() {
				t.Helper()
				err := recover()
				if err != nil {
					if fmt.Sprint(err) != eerr {
						t.Errorf("expected error %q, got %q", eerr, err)
					}
				}
			}()
			out := nr.PathFor(args...)
			if out != eerr {
				t.Errorf("expected error %q for %s, got %q", eerr, args, out)
			}
		}()
	}

	expectError("PathFor requires at least one argument")
	expectError("path not found for [patatas]", "patatas")
	expectError("path name \"posts\" requires 0 arguments. Got 1", "posts", 3)
	expectPath("/posts", "posts")
	expectPath("/posts/3", "post", 3)
	expectPath("/comments/8", &Comment{ID: 8})
	expectPath("/posts/3/comments/8", &Post{ID: 3}, &Comment{ID: 8})
	expectPath("/posts/3/comments/8", "post_comment", &Post{ID: 3}, &Comment{ID: 8})
	expectPath("/posts/3/comments/8", "post_comment", 3, &Comment{ID: 8})
	expectPath("/posts/3/comments/8", "post_comment", "3", &Comment{ID: 8})

}

func TestRouter(t *testing.T) {

	nr := newNamedRoutes()

	// Simple without params
	nr.Add(Route{Path: "/posts", Name: "posts"})
	testRoute(t, nr, "/posts", []any{"posts"})

	// Simple with one param
	nr.Add(Route{Path: "/posts/:id", Name: "post", Models: []any{&Post{}}})
	testRoute(t, nr, "/posts/1",
		args{&Post{ID: 1}},
		args{"post", &Post{ID: 1}},
		args{"post", 1},
		args{"post", "1"},
	)

	// Simple with one param and an action
	nr.Add(Route{
		Path:   "/posts/:post_id/comments",
		Name:   "post_comments",
		Models: []any{&Post{}},
	})

	testRoute(t, nr, "/posts/1/comments",
		args{"post_comments", &User{ID: 1}},
		args{"post_comments", 1},
	)

	// Simple with two params
	nr.Add(Route{
		Path:   "/posts/:post_id/comments/:id",
		Name:   "post_comment",
		Models: []any{&Post{}, &Comment{}},
	})

	testRoute(t, nr, "/posts/1/comments/2",
		args{&Post{ID: 1}, &Comment{ID: 2}},
		args{"post_comment", &Post{ID: 1}, &Comment{ID: 2}},
		args{"post_comment", 1, &Comment{ID: 2}},
		args{"post_comment", 1, 2},
		args{"post_comment", "1", "2"},
	)

}

func testRoute(t *testing.T, nr *namedRoutes, expected string, args ...[]any) {
	t.Helper()
	for _, pathForArgs := range args {
		path := nr.PathFor(pathForArgs...)
		if path != expected {
			t.Fatalf("expected %+v to generate route %s, got %s", pathForArgs, expected, path)
		}
	}
}
