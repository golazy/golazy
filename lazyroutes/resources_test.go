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
	mux := http.NewServeMux()
	Resources(context.Background(), mux, newArticlesController, func(r *Resource[articlesController]) {
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
			mux.ServeHTTP(response, httptest.NewRequest(tt.method, tt.path, nil))
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
			}
			if response.Body.String() != tt.body {
				t.Fatalf("body = %q, want %q", response.Body.String(), tt.body)
			}
		})
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
	mux := http.NewServeMux()
	Resources(context.Background(), mux, newProfilesController, func(r *Resource[profilesController]) {
		r.Path("people")
		r.Param("user")
	})

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/people/guillermo", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if response.Body.String() != "profile guillermo" {
		t.Fatalf("body = %q, want %q", response.Body.String(), "profile guillermo")
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
