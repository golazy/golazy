package lazyview_test

import (
	"fmt"
	"strings"
	"testing/fstest"

	"golazy.dev/lazyview"
	_ "golazy.dev/lazyview/gotmpl"
)

func ExampleViews_Render() {
	views, err := lazyview.New(fstest.MapFS{
		"layouts/app.html.tpl": {Data: []byte(`<main>{{.content}}</main>`)},
		"posts/index.html.tpl": {Data: []byte(`{{ title .name }}{{ partial "tagline" . }}`)},
		"posts/_tagline.html.tpl": {
			Data: []byte(`<p>{{.name}} writes Go.</p>`),
		},
	})
	if err != nil {
		panic(err)
	}
	views.Helper("title", func(name string) string {
		return "Hello, " + name + "."
	})

	var out strings.Builder
	err = views.Render(lazyview.Options{
		Writer:     &out,
		Variables:  map[string]any{"name": "Ada"},
		Controller: "posts",
		Action:     "index",
		UseLayout:  true,
	})
	if err != nil {
		panic(err)
	}

	fmt.Println(out.String())

	// Output:
	// <main>Hello, Ada.<p>Ada writes Go.</p></main>
}
