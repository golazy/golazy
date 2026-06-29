package lazycontrolplane_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"

	"golazy.dev/lazycontrolplane"
)

func Example() {
	databaseReady := false
	plane := lazycontrolplane.New(lazycontrolplane.Config{
		Readiness: []lazycontrolplane.ReadinessCheck{{
			Name: "database",
			Check: func(context.Context) error {
				if !databaseReady {
					return errors.New("connecting")
				}
				return nil
			},
		}},
	})
	plane.Handle("GET /version", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "v1\n")
	}))

	response := httptest.NewRecorder()
	plane.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	fmt.Println(response.Code)

	databaseReady = true
	response = httptest.NewRecorder()
	plane.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	fmt.Print(response.Body.String())

	// Output:
	// 503
	// ready
}
