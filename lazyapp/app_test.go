package lazyapp_test

import (
	"embed"
	"net/http"
	"net/http/httptest"
	"testing"

	"golazy.dev/lazyaction"
	"golazy.dev/lazyapp"
	lazyassets "golazy.dev/lazyassets"
)

//go:embed test_assets/*
var FS embed.FS

type PagesController struct {
}

func (c *PagesController) Index(ctx lazyaction.Context) {
	ctx.WriteString("Hello")
}

func TestLazyApp_Assets(t *testing.T) {

	app := lazyapp.App{
		Files: lazyassets.NewManager(FS, "test_assets"),
	}
	app.Init()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/TEST.md", nil)

	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("Expected 200. Got %d", rec.Code)
	}
	if rec.Body.String() != "Hola" {
		t.Errorf("Expected Hello World. Got: %q", rec.Body.String())
	}

}

func TestLazyApp_Mount(t *testing.T) {

	app := lazyapp.App{}
	app.Init()

	app.Route("* /asdf/*", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello"))
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/asdf", nil)
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("Expected 200. Got %d", rec.Code)
	}
	if rec.Body.String() != "Hello" {
		t.Errorf("Expected Hello World. Got: %q", rec.Body.String())
	}

}

func TestLazyApp_Middleware(t *testing.T) {
	app := lazyapp.App{
		MiddleWares: []lazyapp.Middleware{
			func(h http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					rec := httptest.NewRecorder()
					h.ServeHTTP(rec, r)
					w.Write([]byte("I am the middleware over: " + rec.Body.String() + "!"))
				})
			},
		},
	}

	app.Router.Route("/", func() string {
		return "me"
	})

	app.Init()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("Expected 200. Got %d", rec.Code)
	}
	if rec.Body.String() != "I am the middleware over: me!" {
		t.Errorf("Expected Hello World. Got: %q", rec.Body.String())
	}

}
