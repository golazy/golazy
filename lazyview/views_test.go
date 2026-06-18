package lazyview_test

import (
	"strings"
	"testing"
	"testing/fstest"

	"golazy.dev/lazyview"
	_ "golazy.dev/lazyview/gotmpl"
)

func TestRenderUsesHelpersAndPartials(t *testing.T) {
	views, err := lazyview.New(fstest.MapFS{
		"layouts/app.html.tpl": {Data: []byte(`<main>{{.content}}</main>`)},
		"posts/index.html.tpl": {Data: []byte(`{{ partial "post" . }} {{hello .name}} {{route_name}}`)},
		"posts/_post.html.tpl": {Data: []byte(`<p>{{.name}}</p>`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	views.AddHelpers(map[string]any{
		"hello": func(name string) string {
			return "hello " + name
		},
		"route_name": func(ctx *lazyview.Context) string {
			return ctx.Route.Name
		},
	})
	if err := views.Cache(); err != nil {
		t.Fatal(err)
	}

	var out strings.Builder
	err = views.Render(lazyview.Options{
		Writer:     &out,
		Variables:  map[string]any{"name": "Ada"},
		Route:      lazyview.Route{Name: "posts", Controller: "posts"},
		Controller: "posts",
		Action:     "index",
		UseLayout:  true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if got, want := out.String(), `<main><p>Ada</p> hello Ada posts</main>`; got != want {
		t.Fatalf("rendered body = %q, want %q", got, want)
	}

	out.Reset()
	err = views.Render(lazyview.Options{
		Writer:     &out,
		Variables:  map[string]any{"name": "Ada"},
		Route:      lazyview.Route{Name: "articles", Controller: "posts"},
		Controller: "posts",
		Action:     "index",
		UseLayout:  true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if got, want := out.String(), `<main><p>Ada</p> hello Ada articles</main>`; got != want {
		t.Fatalf("rendered body = %q, want %q", got, want)
	}
}

func TestPartialUsesCurrentDotAsContext(t *testing.T) {
	type postView struct {
		Title string
	}

	views, err := lazyview.New(fstest.MapFS{
		"layouts/app.html.tpl": {Data: []byte(`<main>{{.content}}</main>`)},
		"posts/index.html.tpl": {Data: []byte(`{{range .posts}}{{ partial "post" . }}{{end}}`)},
		"posts/_post.html.tpl": {Data: []byte(`<p>{{.Title}}</p>`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := views.Cache(); err != nil {
		t.Fatal(err)
	}

	var out strings.Builder
	err = views.Render(lazyview.Options{
		Writer: &out,
		Variables: map[string]any{
			"posts": []postView{
				{Title: "First"},
				{Title: "Second"},
			},
		},
		Controller: "posts",
		Action:     "index",
		UseLayout:  true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if got, want := out.String(), `<main><p>First</p><p>Second</p></main>`; got != want {
		t.Fatalf("rendered body = %q, want %q", got, want)
	}
}

func TestPartialFallsBackToAppViews(t *testing.T) {
	views, err := lazyview.New(fstest.MapFS{
		"layouts/app.html.tpl":    {Data: []byte(`<main>{{.content}}</main>`)},
		"posts/index.html.tpl":    {Data: []byte(`{{ partial "shared" . }}`)},
		"app/_shared.html.tpl":    {Data: []byte(`<p>{{.name}}</p>`)},
		"posts/_ignored.html.tpl": {Data: []byte(`ignored`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := views.Cache(); err != nil {
		t.Fatal(err)
	}

	var out strings.Builder
	err = views.Render(lazyview.Options{
		Writer:     &out,
		Variables:  map[string]any{"name": "Ada"},
		Controller: "posts",
		Action:     "index",
		UseLayout:  true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if got, want := out.String(), `<main><p>Ada</p></main>`; got != want {
		t.Fatalf("rendered body = %q, want %q", got, want)
	}
}

func TestPartialReportsTriedViews(t *testing.T) {
	views, err := lazyview.New(fstest.MapFS{
		"layouts/app.html.tpl": {Data: []byte(`<main>{{.content}}</main>`)},
		"posts/index.html.tpl": {Data: []byte(`{{ partial "missing" . }}`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := views.Cache(); err != nil {
		t.Fatal(err)
	}

	var out strings.Builder
	err = views.Render(lazyview.Options{
		Writer:     &out,
		Controller: "posts",
		Action:     "index",
		UseLayout:  true,
	})
	if err == nil {
		t.Fatal("Render succeeded, want missing partial error")
	}
	message := err.Error()
	for _, expected := range []string{
		"posts/_missing.html.tpl",
		"app/_missing.html.tpl",
	} {
		if !strings.Contains(message, expected) {
			t.Fatalf("error = %q, want tried path %q", message, expected)
		}
	}
}

func TestPartialUsesMapArgumentAsContext(t *testing.T) {
	views, err := lazyview.New(fstest.MapFS{
		"layouts/app.html.tpl": {Data: []byte(`<main>{{.content}}</main>`)},
		"posts/index.html.tpl": {Data: []byte(`{{ partial "post" (locals "Grace") }}`)},
		"posts/_post.html.tpl": {Data: []byte(`<p>{{.name}}{{with .site}} {{.}}{{else}} none{{end}}</p>`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	views.AddHelpers(map[string]any{
		"locals": func(name string) map[string]any {
			return map[string]any{"name": name}
		},
	})
	if err := views.Cache(); err != nil {
		t.Fatal(err)
	}

	var out strings.Builder
	err = views.Render(lazyview.Options{
		Writer:     &out,
		Variables:  map[string]any{"name": "Ada", "site": "GoLazy"},
		Controller: "posts",
		Action:     "index",
		UseLayout:  true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if got, want := out.String(), `<main><p>Grace none</p></main>`; got != want {
		t.Fatalf("rendered body = %q, want %q", got, want)
	}
}

func TestRenderFallsBackToAppViews(t *testing.T) {
	views, err := lazyview.New(fstest.MapFS{
		"layouts/app.html.tpl": {Data: []byte(`<main>{{.content}}</main>`)},
		"app/error.html.tpl":   {Data: []byte(`fallback {{.status}}`)},
	})
	if err != nil {
		t.Fatal(err)
	}

	var out strings.Builder
	err = views.Render(lazyview.Options{
		Writer:     &out,
		Variables:  map[string]any{"status": 404},
		Route:      lazyview.Route{Controller: "posts", Action: "show"},
		Controller: "posts",
		Action:     "error",
		UseLayout:  true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if got, want := out.String(), `<main>fallback 404</main>`; got != want {
		t.Fatalf("rendered body = %q, want %q", got, want)
	}
}

func TestNamespacedRenderUsesNestedControllerView(t *testing.T) {
	views, err := lazyview.New(fstest.MapFS{
		"layouts/app.html.tpl":       {Data: []byte(`<main>{{.content}}</main>`)},
		"posts/index.html.tpl":       {Data: []byte(`wrong`)},
		"admin/posts/index.html.tpl": {Data: []byte(`admin`)},
	})
	if err != nil {
		t.Fatal(err)
	}

	var out strings.Builder
	err = views.Render(lazyview.Options{
		Writer:     &out,
		Namespace:  "admin",
		Controller: "posts",
		Action:     "index",
		UseLayout:  true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if got, want := out.String(), `<main>admin</main>`; got != want {
		t.Fatalf("rendered body = %q, want %q", got, want)
	}
}

func TestNamespacedRenderDoesNotFallbackToControllerView(t *testing.T) {
	views, err := lazyview.New(fstest.MapFS{
		"layouts/app.html.tpl": {Data: []byte(`<main>{{.content}}</main>`)},
		"posts/index.html.tpl": {Data: []byte(`wrong`)},
	})
	if err != nil {
		t.Fatal(err)
	}

	var out strings.Builder
	err = views.Render(lazyview.Options{
		Writer:     &out,
		Namespace:  "admin",
		Controller: "posts",
		Action:     "index",
		UseLayout:  true,
	})
	if err == nil {
		t.Fatal("Render succeeded, want missing namespaced view")
	}
	message := err.Error()
	if strings.Contains(message, "Tried: posts/index.html.tpl") || strings.Contains(message, ", posts/index.html.tpl") {
		t.Fatalf("error = %q, should not try non-namespaced controller view", message)
	}
	for _, expected := range []string{
		"admin/posts/index.html.tpl",
		"app/index.html.tpl",
	} {
		if !strings.Contains(message, expected) {
			t.Fatalf("error = %q, want tried path %q", message, expected)
		}
	}
}
