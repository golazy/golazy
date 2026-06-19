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

func TestRedirectDefaultsToFound(t *testing.T) {
	base, response := newRedirectTestBase(t)
	request := httptest.NewRequest(http.MethodGet, "https://example.com/posts/new", nil)
	if err := base.BindRequest(response, request, lazyview.Route{Controller: "posts"}); err != nil {
		t.Fatal(err)
	}

	if err := base.Redirect("/posts"); err != nil {
		t.Fatal(err)
	}

	if response.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusFound)
	}
	if got, want := response.Header().Get("Location"), "/posts"; got != want {
		t.Fatalf("Location = %q, want %q", got, want)
	}
}

func TestRedirectAcceptsSeeOther(t *testing.T) {
	base, response := newRedirectTestBase(t)
	request := httptest.NewRequest(http.MethodPost, "https://example.com/posts", nil)
	if err := base.BindRequest(response, request, lazyview.Route{Controller: "posts"}); err != nil {
		t.Fatal(err)
	}

	if err := base.Redirect("/posts", http.StatusSeeOther); err != nil {
		t.Fatal(err)
	}

	if response.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusSeeOther)
	}
	if got, want := response.Header().Get("Location"), "/posts"; got != want {
		t.Fatalf("Location = %q, want %q", got, want)
	}
}

func TestRedirectRejectsInvalidStatus(t *testing.T) {
	base, response := newRedirectTestBase(t)
	request := httptest.NewRequest(http.MethodGet, "https://example.com/posts", nil)
	if err := base.BindRequest(response, request, lazyview.Route{Controller: "posts"}); err != nil {
		t.Fatal(err)
	}

	if err := base.Redirect("/posts", http.StatusOK); err == nil {
		t.Fatal("expected invalid redirect status error")
	}
	if location := response.Header().Get("Location"); location != "" {
		t.Fatalf("Location = %q, want empty", location)
	}
	if response.Body.Len() != 0 {
		t.Fatalf("body = %q, want empty", response.Body.String())
	}
}

func TestRedirectRejectsPathRelativeLocation(t *testing.T) {
	base, response := newRedirectTestBase(t)
	request := httptest.NewRequest(http.MethodGet, "https://example.com/posts", nil)
	if err := base.BindRequest(response, request, lazyview.Route{Controller: "posts"}); err != nil {
		t.Fatal(err)
	}

	if err := base.Redirect("example.com"); err == nil {
		t.Fatal("expected path-relative redirect error")
	}
	if location := response.Header().Get("Location"); location != "" {
		t.Fatalf("Location = %q, want empty", location)
	}
}

func TestRedirectBackOrToUsesSameHostReferer(t *testing.T) {
	base, response := newRedirectTestBase(t)
	request := httptest.NewRequest(http.MethodGet, "https://example.com/comments", nil)
	request.Header.Set("Referer", "https://example.com/posts/1")
	if err := base.BindRequest(response, request, lazyview.Route{Controller: "comments"}); err != nil {
		t.Fatal(err)
	}

	if err := base.RedirectBackOrTo("/posts"); err != nil {
		t.Fatal(err)
	}

	if got, want := response.Header().Get("Location"), "https://example.com/posts/1"; got != want {
		t.Fatalf("Location = %q, want %q", got, want)
	}
}

func TestRedirectBackOrToUsesFallbackForExternalReferer(t *testing.T) {
	base, response := newRedirectTestBase(t)
	request := httptest.NewRequest(http.MethodGet, "https://example.com/comments", nil)
	request.Header.Set("Referer", "https://evil.example/posts/1")
	if err := base.BindRequest(response, request, lazyview.Route{Controller: "comments"}); err != nil {
		t.Fatal(err)
	}

	if err := base.RedirectBackOrTo("/posts"); err != nil {
		t.Fatal(err)
	}

	if got, want := response.Header().Get("Location"), "/posts"; got != want {
		t.Fatalf("Location = %q, want %q", got, want)
	}
}

func TestURLFromAllowsOnlyInternalLocations(t *testing.T) {
	base, response := newRedirectTestBase(t)
	request := httptest.NewRequest(http.MethodGet, "https://example.com/comments", nil)
	if err := base.BindRequest(response, request, lazyview.Route{Controller: "comments"}); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		location string
		want     string
	}{
		{name: "same host absolute URL", location: "https://example.com/posts", want: "https://example.com/posts"},
		{name: "absolute path", location: "/posts", want: "/posts"},
		{name: "external host", location: "https://evil.example/posts"},
		{name: "path relative", location: "example.com/posts"},
		{name: "protocol relative external host", location: "//evil.example/posts"},
		{name: "invalid control character", location: "/posts\nLocation: https://evil.example"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := base.URLFrom(tt.location); got != tt.want {
				t.Fatalf("URLFrom(%q) = %q, want %q", tt.location, got, tt.want)
			}
		})
	}
}

func newRedirectTestBase(t *testing.T) (Base, *httptest.ResponseRecorder) {
	t.Helper()
	renderer, err := NewRenderer(fstest.MapFS{
		"layouts/app.html.tpl": {Data: []byte(`{{.content}}`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := WithRenderer(context.Background(), renderer)
	base, err := NewBase(ctx)
	if err != nil {
		t.Fatal(err)
	}
	return base, httptest.NewRecorder()
}
