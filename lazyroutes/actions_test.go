package lazyroutes

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"golazy.dev/lazycontroller"
	"golazy.dev/lazydispatch"
	"golazy.dev/lazyview"
	_ "golazy.dev/lazyview/gotmpl"
)

type autoRenderController struct {
	lazycontroller.Base
}

func newAutoRenderController(ctx context.Context) (*autoRenderController, error) {
	base, err := lazycontroller.NewBase(ctx)
	if err != nil {
		return nil, err
	}
	return &autoRenderController{Base: base}, nil
}

func (c *autoRenderController) Index(_ http.ResponseWriter, _ *http.Request) error {
	c.Set("message", "hello")
	return nil
}

func (c *autoRenderController) Write(w http.ResponseWriter, _ *http.Request) error {
	_, err := fmt.Fprint(w, "direct")
	return err
}

func (c *autoRenderController) Broken(w http.ResponseWriter, _ *http.Request) error {
	_, _ = fmt.Fprint(w, "partial")
	return lazycontroller.Error(http.StatusNotFound, errors.New("missing post"))
}

func (c *autoRenderController) Panic(w http.ResponseWriter, _ *http.Request) error {
	_, _ = fmt.Fprint(w, "partial")
	panic("boom")
}

func TestControllerActionAutoRendersDefaultView(t *testing.T) {
	scope := newAutoRenderScope(t)
	scope.Get("/posts", newAutoRenderController, (*autoRenderController).Index)
	handler := lazydispatch.ResponseBuffer().Handler(scope)

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/posts", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if got, want := response.Body.String(), "<main>index hello</main>"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestControllerActionSkipsAutoRenderWhenResponseWasWritten(t *testing.T) {
	scope := newAutoRenderScope(t)
	scope.Get("/write", newAutoRenderController, (*autoRenderController).Write)

	response := httptest.NewRecorder()
	scope.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/write", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if got, want := response.Body.String(), "direct"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestControllerPanicRendersAppFallback(t *testing.T) {
	scope := newAutoRenderScope(t)
	scope.Get("/panic", newAutoRenderController, (*autoRenderController).Panic)
	handler := lazydispatch.ResponseBuffer().Handler(scope)

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/panic", nil))

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	body := response.Body.String()
	if strings.Contains(body, "partial") {
		t.Fatalf("body contains stale partial response: %q", body)
	}
	if !strings.Contains(body, "error 500 Internal Server Error") || !strings.Contains(body, "panic: boom") {
		t.Fatalf("unexpected body: %q", body)
	}
}

func TestControllerErrorUsesStaticFallbackWhenErrorTemplateFails(t *testing.T) {
	renderer, err := lazycontroller.NewRenderer(fstest.MapFS{
		"layouts/app.html.tpl": {Data: []byte(`<main>{{.content}}</main>`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := lazycontroller.WithRenderer(context.Background(), renderer)
	ctx = lazycontroller.WithErrorPages(ctx, fstest.MapFS{
		"500.html": {Data: []byte(`<h1>static 500</h1>`)},
	})
	scope := New(ctx)
	scope.Get("/broken", newAutoRenderController, (*autoRenderController).Broken)
	handler := lazydispatch.ResponseBuffer().Handler(scope)

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/broken", nil))

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	body := response.Body.String()
	if strings.Contains(body, "partial") {
		t.Fatalf("body contains stale partial response: %q", body)
	}
	if got, want := body, "<h1>static 500</h1>"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestControllerHandleErrorCanServeStaticStatusPage(t *testing.T) {
	renderer, err := lazycontroller.NewRenderer(fstest.MapFS{
		"layouts/app.html.tpl": {Data: []byte(`<main>{{.content}}</main>`)},
		"app/error.html.tpl":   {Data: []byte(`dynamic {{.status}}`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := lazycontroller.WithRenderer(context.Background(), renderer)
	ctx = lazycontroller.WithErrorPages(ctx, fstest.MapFS{
		"404.html": {Data: []byte(`<h1>static 404</h1>`)},
		"500.html": {Data: []byte(`<h1>static 500</h1>`)},
	})
	scope := New(ctx)
	scope.Get("/broken", newAutoRenderController, (*autoRenderController).Broken)
	handler := lazydispatch.ResponseBuffer().Handler(scope)

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/broken", nil))

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
	if got, want := response.Body.String(), "<h1>static 404</h1>"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestControllerErrorResetsBufferAndRendersAppFallback(t *testing.T) {
	scope := newAutoRenderScope(t)
	scope.Get("/broken", newAutoRenderController, (*autoRenderController).Broken)
	handler := lazydispatch.ResponseBuffer().Handler(scope)

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/broken", nil))

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
	body := response.Body.String()
	if strings.Contains(body, "partial") {
		t.Fatalf("body contains stale partial response: %q", body)
	}
	if !strings.Contains(body, "error 404 Not Found") || !strings.Contains(body, "missing post") {
		t.Fatalf("unexpected body: %q", body)
	}
}

func newAutoRenderScope(t *testing.T) *Scope {
	t.Helper()
	renderer, err := lazycontroller.NewRenderer(fstest.MapFS{
		"layouts/app.html.tpl": {Data: []byte(`<main>{{.content}}</main>`)},
		"auto_render/index.html.tpl": {
			Data: []byte(`index {{.message}}`),
		},
		"app/error.html.tpl": {Data: []byte(`error {{.status}} {{.statusText}} {{.error}}`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := lazycontroller.WithRenderer(context.Background(), renderer)
	ctx = lazycontroller.WithRoute(ctx, lazyview.Route{Controller: "auto_render"})
	return New(ctx)
}
