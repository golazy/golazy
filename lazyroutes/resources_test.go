package lazyroutes

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"golazy.dev/lazycontroller"
)

type articlesController struct{}

func newArticlesController(context.Context) (*articlesController, error) {
	return &articlesController{}, nil
}

func (c *articlesController) Index(w http.ResponseWriter, _ *http.Request) error {
	_, err := fmt.Fprint(w, "index")
	return err
}

func (c *articlesController) New(w http.ResponseWriter, _ *http.Request) error {
	_, err := fmt.Fprint(w, "new")
	return err
}

func (c *articlesController) Create(w http.ResponseWriter, _ *http.Request) error {
	_, err := fmt.Fprint(w, "create")
	return err
}

func (c *articlesController) Show(w http.ResponseWriter, r *http.Request) error {
	_, err := fmt.Fprintf(w, "show %s", r.PathValue("article_id"))
	return err
}

func (c *articlesController) Edit(w http.ResponseWriter, r *http.Request) error {
	_, err := fmt.Fprintf(w, "edit %s", r.PathValue("article_id"))
	return err
}

func (c *articlesController) Update(w http.ResponseWriter, r *http.Request) error {
	_, err := fmt.Fprintf(w, "update %s", r.PathValue("article_id"))
	return err
}

func (c *articlesController) Delete(w http.ResponseWriter, r *http.Request) error {
	_, err := fmt.Fprintf(w, "delete %s", r.PathValue("article_id"))
	return err
}

func (c *articlesController) Search(w http.ResponseWriter, _ *http.Request) error {
	_, err := fmt.Fprint(w, "search")
	return err
}

func (c *articlesController) Preview(w http.ResponseWriter, r *http.Request) error {
	_, err := fmt.Fprintf(w, "preview %s", r.PathValue("article_id"))
	return err
}

func TestResourcesRegistersRESTAndCustomRoutes(t *testing.T) {
	scope := New(context.Background())
	scope.Resources(newArticlesController, func(r *Resource) {
		r.Get("search", (*articlesController).Search)
		r.MemberGet("preview", (*articlesController).Preview)
	})

	tests := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/articles", "index"},
		{http.MethodGet, "/articles/new", "new"},
		{http.MethodPost, "/articles", "create"},
		{http.MethodGet, "/articles/hello", "show hello"},
		{http.MethodGet, "/articles/hello/edit", "edit hello"},
		{http.MethodPatch, "/articles/hello", "update hello"},
		{http.MethodPut, "/articles/hello", "update hello"},
		{http.MethodDelete, "/articles/hello", "delete hello"},
		{http.MethodGet, "/articles/search", "search"},
		{http.MethodGet, "/articles/hello/preview", "preview hello"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			response := httptest.NewRecorder()
			scope.ServeHTTP(response, httptest.NewRequest(tt.method, tt.path, nil))
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
			}
			if response.Body.String() != tt.body {
				t.Fatalf("body = %q, want %q", response.Body.String(), tt.body)
			}
		})
	}

	routeByPathAndMethod := func(method, path string) (Route, bool) {
		for _, route := range scope.Routes {
			if route.Method == method && route.Path == path {
				return route, true
			}
		}
		return Route{}, false
	}
	if route, ok := routeByPathAndMethod(http.MethodGet, "/articles/{article_id}"); !ok || route.Name != "article" {
		t.Fatalf("expected article route for GET /articles/{article_id}, got %#v (found=%v)", route, ok)
	}
	if route, ok := routeByPathAndMethod(http.MethodGet, "/articles/search"); !ok || route.Name != "search_articles" {
		t.Fatalf("expected search_articles route for GET /articles/search, got %#v (found=%v)", route, ok)
	}
	if route, ok := routeByPathAndMethod(http.MethodGet, "/articles/{article_id}/preview"); !ok || route.Name != "preview_article" {
		t.Fatalf("expected preview_article route for GET /articles/{article_id}/preview, got %#v (found=%v)", route, ok)
	}
	if route, ok := routeByPathAndMethod(http.MethodGet, "/articles/{article_id}"); !ok || !route.NamedParams["article_id"] {
		t.Fatalf("expected route for article_id param, got %#v (found=%v)", route, ok)
	}
}

type profilesController struct{}

func newProfilesController(context.Context) (*profilesController, error) {
	return &profilesController{}, nil
}

func (c *profilesController) Show(w http.ResponseWriter, r *http.Request) error {
	_, err := fmt.Fprintf(w, "profile %s", r.PathValue("user"))
	return err
}

func TestResourcesSupportsOverrides(t *testing.T) {
	scope := New(context.Background())
	scope.Resources(newProfilesController, func(r *Resource) {
		r.Path("people")
		r.Param("user")
	})

	response := httptest.NewRecorder()
	scope.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/people/guillermo", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if response.Body.String() != "profile guillermo" {
		t.Fatalf("body = %q, want %q", response.Body.String(), "profile guillermo")
	}
}

type articleModel struct {
	Slug string
}

func (m articleModel) RouteParam() string {
	return m.Slug
}

func TestResourcesMapModelsToRESTPaths(t *testing.T) {
	scope := New(context.Background())
	scope.Resources(newArticlesController, func(r *Resource) {
		r.Model(articleModel{})
	})

	create, err := scope.PathForModel(articleModel{}, "create")
	if err != nil {
		t.Fatal(err)
	}
	if create != "/articles" {
		t.Fatalf("create path = %q, want /articles", create)
	}

	update, err := scope.PathForModel(articleModel{Slug: "hello world"}, "update")
	if err != nil {
		t.Fatal(err)
	}
	if update != "/articles/hello%20world" {
		t.Fatalf("update path = %q, want /articles/hello%%20world", update)
	}

	deletePath, err := scope.PathForModel(&articleModel{Slug: "hello"}, "delete")
	if err != nil {
		t.Fatal(err)
	}
	if deletePath != "/articles/hello" {
		t.Fatalf("delete path = %q, want /articles/hello", deletePath)
	}
}

func TestResourcesUseNamespaceScope(t *testing.T) {
	scope := New(context.Background())

	scope.Namespace("admin", func(admin *Scope) {
		admin.Resources(newArticlesController, func(articles *Resource) {
			articles.Model(articleModel{})
		})
	})

	response := httptest.NewRecorder()
	scope.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/admin/articles/hello", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if response.Body.String() != "show hello" {
		t.Fatalf("body = %q, want %q", response.Body.String(), "show hello")
	}

	for _, route := range scope.Routes {
		if route.Method == http.MethodGet && route.Path == "/admin/articles/{article_id}" {
			if route.Name != "admin_article" {
				t.Fatalf("route.Name = %q, want %q", route.Name, "admin_article")
			}
			if route.Namespace != "admin" {
				t.Fatalf("route.Namespace = %q, want %q", route.Namespace, "admin")
			}
			if !route.NamedParams["article_id"] {
				t.Fatalf("route.NamedParams = %#v, want article_id", route.NamedParams)
			}
			path, err := scope.PathForModel(articleModel{Slug: "hello"}, "update")
			if err != nil {
				t.Fatal(err)
			}
			if path != "/admin/articles/hello" {
				t.Fatalf("PathForModel update = %q, want /admin/articles/hello", path)
			}
			return
		}
	}
	t.Fatalf("admin article route not found in %#v", scope.Routes)
}

type votesController struct{}

func newVotesController(context.Context) (*votesController, error) {
	return &votesController{}, nil
}

func (c *votesController) Create(w http.ResponseWriter, r *http.Request) error {
	_, err := fmt.Fprintf(w, "vote %s", r.PathValue("article_id"))
	return err
}

func (c *votesController) Show(w http.ResponseWriter, r *http.Request) error {
	_, err := fmt.Fprintf(w, "vote %s %s", r.PathValue("article_id"), r.PathValue("vote_id"))
	return err
}

func TestResourcesSupportsNestedResources(t *testing.T) {
	scope := New(context.Background())
	scope.Resources(newArticlesController, func(articles *Resource) {
		articles.Resources(newVotesController)
	})

	tests := []struct {
		method    string
		path      string
		routePath string
		body      string
		name      string
	}{
		{http.MethodPost, "/articles/hello/votes", "/articles/{article_id}/votes", "vote hello", "votes_article"},
		{http.MethodGet, "/articles/hello/votes/99", "/articles/{article_id}/votes/{vote_id}", "vote hello 99", "vote_article"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			response := httptest.NewRecorder()
			scope.ServeHTTP(response, httptest.NewRequest(tt.method, tt.path, nil))
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
			}
			if response.Body.String() != tt.body {
				t.Fatalf("body = %q, want %q", response.Body.String(), tt.body)
			}

			for _, route := range scope.Routes {
				if route.Method == tt.method && route.Path == tt.routePath {
					if route.Name != tt.name {
						t.Fatalf("route.Name = %q, want %q", route.Name, tt.name)
					}
					if !route.NamedParams["article_id"] {
						t.Fatalf("route.NamedParams = %#v, want article_id", route.NamedParams)
					}
					return
				}
			}
			t.Fatalf("route %s %s not found in %#v", tt.method, tt.path, scope.Routes)
		})
	}

	path, err := scope.PathFor("votes_article", "hello")
	if err != nil {
		t.Fatal(err)
	}
	if path != "/articles/hello/votes" {
		t.Fatalf("PathFor votes_article = %q, want /articles/hello/votes", path)
	}
}

func TestHandleConvertsHTTPErrorStatus(t *testing.T) {
	handler := Handle(func(http.ResponseWriter, *http.Request) error {
		return lazycontroller.Error(http.StatusTeapot, errors.New("short and stout"))
	})

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusTeapot {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusTeapot)
	}
}
