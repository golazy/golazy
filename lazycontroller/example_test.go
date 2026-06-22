package lazycontroller_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing/fstest"

	"golazy.dev/lazycontroller"
	"golazy.dev/lazyview"
	_ "golazy.dev/lazyview/gotmpl"
)

func ExampleBase_Render() {
	renderer, err := lazycontroller.NewRenderer(fstest.MapFS{
		"layouts/app.html.tpl": {Data: []byte("<main>{{.content}}</main>")},
		"posts/index.html.tpl": {Data: []byte("<h1>{{.Title}}</h1>")},
	})
	if err != nil {
		panic(err)
	}

	ctx := lazycontroller.WithRenderer(context.Background(), renderer)
	base, err := lazycontroller.NewBase(ctx, "posts")
	if err != nil {
		panic(err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/posts", nil)
	err = base.BindRequest(recorder, request, lazyview.Route{
		Controller: "posts",
		Action:     "Index",
	})
	if err != nil {
		panic(err)
	}

	base.Set("Title", "Posts")
	if err := base.Render(""); err != nil {
		panic(err)
	}

	fmt.Println(recorder.Body.String())
	// Output: <main><h1>Posts</h1></main>
}
