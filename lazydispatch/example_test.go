package lazydispatch_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing/fstest"

	"golazy.dev/lazydispatch"
)

type exampleRouter struct{}

func (exampleRouter) HandlesPath(path string) bool {
	return path == "/hello"
}

func (exampleRouter) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	_, _ = fmt.Fprint(w, "hello")
}

func ExampleDispatcher() {
	router := exampleRouter{}

	dispatcher := lazydispatch.NewDispatcher()
	dispatcher.Use(lazydispatch.RouteOnly(router, lazydispatch.ETag()))
	dispatcher.Use(lazydispatch.Router(router))
	dispatcher.Use(lazydispatch.Static(fstest.MapFS{
		"app.css": {Data: []byte("body { color: black; }")},
	}))

	request := httptest.NewRequest(http.MethodGet, "/hello", nil)
	response := httptest.NewRecorder()
	dispatcher.ServeHTTP(response, request)

	fmt.Println(response.Code)
	fmt.Println(response.Body.String())
	fmt.Println(response.Header().Get("ETag") != "")

	request = httptest.NewRequest(http.MethodGet, "/app.css", nil)
	response = httptest.NewRecorder()
	dispatcher.ServeHTTP(response, request)

	fmt.Println(response.Code)
	fmt.Println(response.Body.String())

	// Output:
	// 200
	// hello
	// true
	// 200
	// body { color: black; }
}
