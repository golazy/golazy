package lazyroutes_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"

	"golazy.dev/lazyroutes"
)

func Example() {
	router := lazyroutes.New(context.Background())
	router.HandleFunc(http.MethodGet, "/health", func(w http.ResponseWriter, r *http.Request) error {
		fmt.Fprint(w, "ok")
		return nil
	})

	path, err := router.PathFor("health")
	if err != nil {
		panic(err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	router.ServeHTTP(recorder, request)

	fmt.Println(path, recorder.Body.String())
	// Output: /health ok
}
