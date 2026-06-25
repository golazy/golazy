package lazycontroller

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	"golazy.dev/lazyview"
	_ "golazy.dev/lazyview/gotmpl"
)

type viewValueContextKey struct{}

func TestSetLaterStartsImmediatelyAndStoresValuer(t *testing.T) {
	base := newViewValueTestBase(t, fstest.MapFS{
		"layouts/app.html.tpl": {Data: []byte(`{{.content}}`)},
		"posts/show.html.tpl":  {Data: []byte(`{{.name.Value}}`)},
	})
	started := make(chan struct{})
	release := make(chan struct{})

	valuer := base.SetLater("name", func() (string, error) {
		close(started)
		<-release
		return "Ada", nil
	})
	if base.data["name"] != valuer {
		t.Fatal("SetLater did not store the returned Valuer")
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("SetLater did not start loader")
	}

	close(release)
	value, err := valuer.Value()
	if err != nil {
		t.Fatal(err)
	}
	if value != "Ada" {
		t.Fatalf("Value = %q, want Ada", value)
	}
}

func TestSetWhenNeededLoadsOnlyOnFirstValue(t *testing.T) {
	base := newViewValueTestBase(t, fstest.MapFS{
		"layouts/app.html.tpl": {Data: []byte(`{{.content}}`)},
		"posts/show.html.tpl":  {Data: []byte(`{{.name.Value}}`)},
	})
	var calls int64
	valuer := base.SetWhenNeeded("name", func() (string, error) {
		atomic.AddInt64(&calls, 1)
		return "Ada", nil
	})
	if calls != 0 {
		t.Fatalf("loader calls = %d, want 0 before Value", calls)
	}

	for range 2 {
		value, err := valuer.Value()
		if err != nil {
			t.Fatal(err)
		}
		if value != "Ada" {
			t.Fatalf("Value = %q, want Ada", value)
		}
	}
	if calls != 1 {
		t.Fatalf("loader calls = %d, want 1", calls)
	}
}

func TestDeferredViewValueRendersInTemplate(t *testing.T) {
	base := newViewValueTestBase(t, fstest.MapFS{
		"layouts/app.html.tpl": {Data: []byte(`<main>{{.content}}</main>`)},
		"posts/show.html.tpl":  {Data: []byte(`<p>{{.name.Value}}</p>`)},
	})
	base.SetWhenNeeded("name", func() (string, error) {
		return "Ada", nil
	})

	if err := base.Render("show"); err != nil {
		t.Fatal(err)
	}
	if body := base.writer.(*httptest.ResponseRecorder).Body.String(); !strings.Contains(body, `<main><p>Ada</p></main>`) {
		t.Fatalf("body = %s, want rendered deferred value", body)
	}
}

func TestDeferredViewValueErrorFailsRender(t *testing.T) {
	base := newViewValueTestBase(t, fstest.MapFS{
		"layouts/app.html.tpl": {Data: []byte(`{{.content}}`)},
		"posts/show.html.tpl":  {Data: []byte(`{{.name.Value}}`)},
	})
	base.SetWhenNeeded("name", func() (string, error) {
		return "", errors.New("load name")
	})

	err := base.Render("show")
	if err == nil || !strings.Contains(err.Error(), "load name") {
		t.Fatalf("Render error = %v, want loader error", err)
	}
}

func TestDeferredViewValueReceivesRequestContext(t *testing.T) {
	base := newViewValueTestBase(t, fstest.MapFS{
		"layouts/app.html.tpl": {Data: []byte(`{{.content}}`)},
		"posts/show.html.tpl":  {Data: []byte(`{{.name.Value}}`)},
	})
	valuer := base.SetWhenNeeded("name", func(ctx context.Context) (string, error) {
		value, _ := ctx.Value(viewValueContextKey{}).(string)
		return value, nil
	})

	value, err := valuer.Value()
	if err != nil {
		t.Fatal(err)
	}
	if value != "request" {
		t.Fatalf("Value = %q, want request context value", value)
	}
}

func TestDeferredViewValueReportsInvalidLoader(t *testing.T) {
	base := newViewValueTestBase(t, fstest.MapFS{
		"layouts/app.html.tpl": {Data: []byte(`{{.content}}`)},
		"posts/show.html.tpl":  {Data: []byte(`{{.name.Value}}`)},
	})
	valuer := base.SetWhenNeeded("name", "not a loader")
	if _, err := valuer.Value(); err == nil || !strings.Contains(err.Error(), "loader must be a function") {
		t.Fatalf("Value error = %v, want invalid loader", err)
	}
}

func newViewValueTestBase(t *testing.T, files fstest.MapFS) Base {
	t.Helper()
	renderer, err := NewRenderer(files)
	if err != nil {
		t.Fatal(err)
	}
	ctx := WithRenderer(context.Background(), renderer)
	base, err := NewBase(ctx)
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	requestContext := context.WithValue(context.Background(), viewValueContextKey{}, "request")
	request := httptest.NewRequest(http.MethodGet, "/posts/1", nil).WithContext(requestContext)
	if err := base.BindRequest(response, request, lazyview.Route{Controller: "posts", Action: "show"}); err != nil {
		t.Fatal(err)
	}
	return base
}
