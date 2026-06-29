package lazyturbo_test

import (
	"fmt"
	"strings"
	"testing/fstest"

	"golazy.dev/lazyturbo"
	"golazy.dev/lazyview"
	_ "golazy.dev/lazyview/gotmpl"
)

func ExampleHelpers() {
	views, err := lazyview.New(fstest.MapFS{
		"layouts/app.html.tpl": {
			Data: []byte(`{{.content}}`),
		},
		"posts/index.html.tpl": {
			Data: []byte(`{{ turbo_frame "post" . (turbo_src "/posts/1") (turbo_loading "lazy") }}`),
		},
		"posts/_post_frame.html.tpl": {
			Data: []byte(`<h1>{{.title}}</h1>`),
		},
	})
	if err != nil {
		panic(err)
	}
	views.AddHelpers(lazyturbo.Helpers())
	if err := views.Cache(); err != nil {
		panic(err)
	}

	var out strings.Builder
	err = views.Render(lazyview.Options{
		Writer:     &out,
		Variables:  map[string]any{"title": "Hello"},
		Controller: "posts",
		Action:     "index",
		UseLayout:  false,
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(out.String())

	// Output:
	// <turbo-frame id="post" src="/posts/1" loading="lazy"><h1>Hello</h1></turbo-frame>
}
