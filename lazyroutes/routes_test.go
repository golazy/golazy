package lazyroutes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"golazy.dev/lazycontroller"
)

func TestScopeRegistersRouteMetadataAndContext(t *testing.T) {
	scope := New(context.Background())
	scope.HandleFunc("GET", "/articles/{post_id}", func(w http.ResponseWriter, r *http.Request) error {
		route, values, ok := RouteFromRequest(r)
		if !ok {
			t.Fatalf("route not found in request context")
		}
		if route.Path != "/articles/{post_id}" {
			t.Fatalf("route.Path = %q, want %q", route.Path, "/articles/{post_id}")
		}
		if route.NamedParams == nil || !route.NamedParams["post_id"] {
			t.Fatalf("named params = %#v, want post_id=true", route.NamedParams)
		}
		if !reflect.DeepEqual(values, map[string]string{"post_id": "42"}) {
			t.Fatalf("values = %#v, want %#v", values, map[string]string{"post_id": "42"})
		}
		w.WriteHeader(http.StatusOK)
		return nil
	})

	if len(scope.Routes) != 1 {
		t.Fatalf("len(scope.Routes) = %d, want 1", len(scope.Routes))
	}
	if scope.Routes[0].Name != "articles" {
		t.Fatalf("route.Name = %q, want %q", scope.Routes[0].Name, "articles")
	}

	response := httptest.NewRecorder()
	scope.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/articles/42", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
}

func TestScopeStoresUserFacingRootRoute(t *testing.T) {
	scope := New(context.Background())
	scope.HandleFunc("GET", "/", func(w http.ResponseWriter, _ *http.Request) error {
		w.WriteHeader(http.StatusOK)
		return nil
	})

	if len(scope.Routes) != 1 {
		t.Fatalf("len(scope.Routes) = %d, want 1", len(scope.Routes))
	}
	if scope.Routes[0].Path != "/" {
		t.Fatalf("scope.Routes[0].Path = %q, want /", scope.Routes[0].Path)
	}
	if scope.Routes[0].NamedParams == nil {
		t.Fatalf("scope.Routes[0].NamedParams = nil, want empty map")
	}

	response := httptest.NewRecorder()
	scope.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	response = httptest.NewRecorder()
	scope.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/missing", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func TestScopeRegistersRouteMetadataWithMultipleParams(t *testing.T) {
	scope := New(context.Background())

	scope.HandleFunc("GET", "/articles/{article_id}/comments/{comment_id}", func(w http.ResponseWriter, r *http.Request) error {
		route, values, ok := RouteFromRequest(r)
		if !ok {
			t.Fatalf("route not found in request context")
		}
		if !route.NamedParams["article_id"] || !route.NamedParams["comment_id"] {
			t.Fatalf("named params = %#v, want article_id and comment_id", route.NamedParams)
		}
		if !reflect.DeepEqual(values, map[string]string{"article_id": "42", "comment_id": "99"}) {
			t.Fatalf("values = %#v, want %#v", values, map[string]string{"article_id": "42", "comment_id": "99"})
		}
		w.WriteHeader(http.StatusOK)
		return nil
	})

	response := httptest.NewRecorder()
	scope.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/articles/42/comments/99", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
}

func TestScopeStripsFormatSuffixBeforeRouting(t *testing.T) {
	scope := New(context.Background())

	scope.HandleFunc("GET", "/articles/{article_id}", func(w http.ResponseWriter, r *http.Request) error {
		route, values, ok := RouteFromRequest(r)
		if !ok {
			t.Fatalf("route not found in request context")
		}
		if got, want := r.URL.Path, "/articles/42"; got != want {
			t.Fatalf("URL.Path = %q, want %q", got, want)
		}
		if got, want := r.PathValue("article_id"), "42"; got != want {
			t.Fatalf("PathValue(article_id) = %q, want %q", got, want)
		}
		if got, want := values["article_id"], "42"; got != want {
			t.Fatalf("route value article_id = %q, want %q", got, want)
		}
		if got, want := lazycontroller.FormatFromRequest(r), lazycontroller.JSON; got != want {
			t.Fatalf("FormatFromRequest = %q, want %q", got, want)
		}
		if route.Path != "/articles/{article_id}" {
			t.Fatalf("route.Path = %q, want %q", route.Path, "/articles/{article_id}")
		}
		w.WriteHeader(http.StatusOK)
		return nil
	})

	response := httptest.NewRecorder()
	scope.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/articles/42.json", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
}

func TestScopeHandlesCollectionFormatSuffix(t *testing.T) {
	scope := New(context.Background())
	scope.HandleFunc("GET", "/articles", func(w http.ResponseWriter, r *http.Request) error {
		if got, want := r.URL.Path, "/articles"; got != want {
			t.Fatalf("URL.Path = %q, want %q", got, want)
		}
		if got, want := lazycontroller.FormatFromRequest(r), lazycontroller.HTML; got != want {
			t.Fatalf("FormatFromRequest = %q, want %q", got, want)
		}
		w.WriteHeader(http.StatusOK)
		return nil
	})

	if !scope.HandlesPath("/articles.html") {
		t.Fatalf("scope should handle /articles.html")
	}

	response := httptest.NewRecorder()
	scope.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/articles.html", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
}

func TestScopeNamespacePrefixesPathNameAndRouteContext(t *testing.T) {
	scope := New(context.Background())

	scope.Namespace("admin", func(admin *Scope) {
		admin.HandleFunc("GET", "/articles/{article_id}", func(w http.ResponseWriter, r *http.Request) error {
			route, values, ok := RouteFromRequest(r)
			if !ok {
				t.Fatalf("route not found in request context")
			}
			if route.Path != "/admin/articles/{article_id}" {
				t.Fatalf("route.Path = %q, want %q", route.Path, "/admin/articles/{article_id}")
			}
			if route.Name != "admin_articles" {
				t.Fatalf("route.Name = %q, want %q", route.Name, "admin_articles")
			}
			if route.Namespace != "admin" {
				t.Fatalf("route.Namespace = %q, want %q", route.Namespace, "admin")
			}
			if !reflect.DeepEqual(values, map[string]string{"article_id": "42"}) {
				t.Fatalf("values = %#v, want %#v", values, map[string]string{"article_id": "42"})
			}
			w.WriteHeader(http.StatusOK)
			return nil
		})
	})

	if len(scope.Routes) != 1 {
		t.Fatalf("len(scope.Routes) = %d, want 1", len(scope.Routes))
	}
	if scope.Routes[0].Path != "/admin/articles/{article_id}" {
		t.Fatalf("scope.Routes[0].Path = %q, want %q", scope.Routes[0].Path, "/admin/articles/{article_id}")
	}

	response := httptest.NewRecorder()
	scope.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/admin/articles/42", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
}

func TestScopeNamespacePrefixesRootRouteName(t *testing.T) {
	scope := New(context.Background())

	scope.Namespace("admin", func(admin *Scope) {
		admin.HandleFunc("GET", "/", func(w http.ResponseWriter, _ *http.Request) error {
			w.WriteHeader(http.StatusOK)
			return nil
		})
	})

	if len(scope.Routes) != 1 {
		t.Fatalf("len(scope.Routes) = %d, want 1", len(scope.Routes))
	}
	if route := scope.Routes[0]; route.Name != "admin_root" {
		t.Fatalf("route.Name = %q, want %q", route.Name, "admin_root")
	}
}

func TestScopePathAndAsComposeRouteMetadata(t *testing.T) {
	scope := New(context.Background())

	account := scope.Path("accounts/{account_id}").As("account")
	account.HandleFunc("GET", "/posts/{post_id}", func(w http.ResponseWriter, r *http.Request) error {
		route, values, ok := RouteFromRequest(r)
		if !ok {
			t.Fatalf("route not found in request context")
		}
		if route.Path != "/accounts/{account_id}/posts/{post_id}" {
			t.Fatalf("route.Path = %q, want %q", route.Path, "/accounts/{account_id}/posts/{post_id}")
		}
		if route.Name != "account_posts" {
			t.Fatalf("route.Name = %q, want %q", route.Name, "account_posts")
		}
		if !route.NamedParams["account_id"] || !route.NamedParams["post_id"] {
			t.Fatalf("route.NamedParams = %#v, want account_id and post_id", route.NamedParams)
		}
		if !reflect.DeepEqual(values, map[string]string{"account_id": "7", "post_id": "42"}) {
			t.Fatalf("values = %#v, want %#v", values, map[string]string{"account_id": "7", "post_id": "42"})
		}
		w.WriteHeader(http.StatusOK)
		return nil
	})

	response := httptest.NewRecorder()
	scope.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/accounts/7/posts/42", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
}

func TestScopeHandlesPathIgnoresMethod(t *testing.T) {
	scope := New(context.Background())
	scope.HandleFunc("GET", "/posts/{post_id}", func(w http.ResponseWriter, _ *http.Request) error {
		w.WriteHeader(http.StatusOK)
		return nil
	})

	if !scope.HandlesPath("/posts/hello") {
		t.Fatalf("scope should handle /posts/hello")
	}
	if scope.HandlesPath("/posts/hello/comments") {
		t.Fatalf("scope should not handle /posts/hello/comments")
	}
}
