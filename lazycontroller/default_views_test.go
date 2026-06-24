package lazycontroller

import (
	"context"
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golazy.dev/lazyview"
	_ "golazy.dev/lazyview/gotmpl"
)

func TestDefaultViewsRenderBaseHandleError(t *testing.T) {
	views, err := DefaultViews()
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{"layouts/app.html.tpl", "app/error.html.tpl"} {
		if _, err := fs.Stat(views, file); err != nil {
			t.Fatalf("default view %s: %v", file, err)
		}
	}

	renderer, err := NewRenderer(views)
	if err != nil {
		t.Fatal(err)
	}
	ctx := WithRenderer(context.Background(), renderer)
	base, err := NewBase(ctx)
	if err != nil {
		t.Fatal(err)
	}

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/missing", nil)
	if err := base.BindRequest(response, request, lazyview.Route{Controller: "posts", Action: "Show"}); err != nil {
		t.Fatal(err)
	}
	if err := base.HandleError(response, request, Error(http.StatusNotFound, errors.New("missing post"))); err != nil {
		t.Fatal(err)
	}

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
	body := response.Body.String()
	expected := []string{"GoLazy", "404 Not Found", "glz-error"}
	if DetailErrors(ctx) {
		expected = append(expected, "missing post")
	} else {
		expected = append(expected, "The request could not be completed")
	}
	for _, want := range expected {
		if !strings.Contains(body, want) {
			t.Fatalf("body does not contain %q:\n%s", want, body)
		}
	}
	if !DetailErrors(ctx) && strings.Contains(body, "missing post") {
		t.Fatalf("body exposed production error detail:\n%s", body)
	}
}

func TestDefaultViewsRenderErrorDetailWhenEnabled(t *testing.T) {
	views, err := DefaultViews()
	if err != nil {
		t.Fatal(err)
	}
	renderer, err := NewRenderer(views)
	if err != nil {
		t.Fatal(err)
	}
	ctx := WithDetailErrors(WithRenderer(context.Background(), renderer))
	base, err := NewBase(ctx)
	if err != nil {
		t.Fatal(err)
	}

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/missing", nil)
	if err := base.BindRequest(response, request, lazyview.Route{Controller: "posts", Action: "Show"}); err != nil {
		t.Fatal(err)
	}
	if err := base.HandleError(response, request, Error(http.StatusNotFound, errors.New("missing post"))); err != nil {
		t.Fatal(err)
	}

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
	if body := response.Body.String(); !strings.Contains(body, "missing post") {
		t.Fatalf("body does not contain detail error:\n%s", body)
	}
}
