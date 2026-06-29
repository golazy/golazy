package lazyapp_test

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing/fstest"

	"golazy.dev/lazyapp"
	"golazy.dev/lazycontroller"
	"golazy.dev/lazyroutes"
	_ "golazy.dev/lazyview/gotmpl"
)

type examplePagesController struct {
	lazycontroller.Base
}

func newExamplePagesController(ctx context.Context) (*examplePagesController, error) {
	base, err := lazycontroller.NewBase(ctx, "pages")
	if err != nil {
		return nil, err
	}
	return &examplePagesController{Base: base}, nil
}

func (c *examplePagesController) Index(http.ResponseWriter, *http.Request) error {
	c.Set("Title", "Hello from lazyapp")
	return nil
}

func ExampleNew() {
	public := fstest.MapFS{
		"styles.css": {Data: []byte("body { font-family: sans-serif; }")},
	}
	views := fstest.MapFS{
		"layouts/app.html.tpl": {Data: []byte("<main>{{.content}}</main>")},
		"pages/index.html.tpl": {Data: []byte("<h1>{{.Title}}</h1>")},
	}

	app := lazyapp.New(lazyapp.Config{
		Name: "example",
		Drawer: func(router *lazyroutes.Scope) {
			router.Get("/", newExamplePagesController, (*examplePagesController).Index)
		},
		Public: func() (fs.FS, error) { return public, nil },
		Views:  func() (fs.FS, error) { return views, nil },
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	app.ServeHTTP(recorder, request)

	fmt.Println(recorder.Body.String())
	// Output: <main><h1>Hello from lazyapp</h1></main>
}
