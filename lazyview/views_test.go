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
		"posts/index.html.tpl": {Data: []byte(`{{partial "post"}} {{hello .name}} {{route_name}}`)},
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
}
