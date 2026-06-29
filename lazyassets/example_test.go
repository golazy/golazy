package lazyassets_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing/fstest"

	"golazy.dev/lazyassets"
)

func ExampleRegistry() {
	registry := lazyassets.New()
	err := registry.AddFS(fstest.MapFS{
		"styles.css": {Data: []byte("body { color: black; }")},
	})
	if err != nil {
		panic(err)
	}

	assetPath, err := registry.Path("/styles.css")
	if err != nil {
		panic(err)
	}
	fmt.Println(assetPath[:8] == "/styles-")

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/styles.css", nil)
	registry.ServeHTTP(response, request)

	fmt.Println(response.Code)
	fmt.Println(response.Header().Get("Content-Type"))
	fmt.Println(response.Body.String())

	// Output:
	// true
	// 200
	// text/css; charset=utf-8
	// body { color: black; }
}
