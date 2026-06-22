package lazyapp_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"

	"golazy.dev/lazyapp"
	"golazy.dev/lazyroutes"
)

type exampleHomeController struct{}

func newExampleHomeController(context.Context) (*exampleHomeController, error) {
	return &exampleHomeController{}, nil
}

func (c *exampleHomeController) Index(w http.ResponseWriter, r *http.Request) error {
	fmt.Fprint(w, "hello")
	return nil
}

func ExampleNew() {
	app := lazyapp.New(lazyapp.Config{
		Name: "example",
		Drawer: func(router *lazyroutes.Scope) {
			router.Get("/", newExampleHomeController, (*exampleHomeController).Index)
		},
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	app.ServeHTTP(recorder, request)

	fmt.Println(recorder.Body.String())
	// Output: hello
}
