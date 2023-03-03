package lazyaction

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func newExpect(t *testing.T, r *Dispatcher) func(path, expected string) {
	t.Helper()
	return func(path, expected string) {
		t.Helper()
		req, err := http.NewRequest("GET", path, nil)
		if err != nil {
			t.Fatal(err)
		}
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Body.String() != expected {
			t.Errorf("expected %q to return %q got %q", path, expected, rr.Body.String())
		}
	}
}

func TestRouterRoute(t *testing.T) {

	router := Dispatcher{}

	expect := newExpect(t, &router)

	router.Route("/:path", ActionHandler)
	router.Route("/hi", StringHandler("hi"))
	expect("/root", "root")
	expect("/hi", "hi")

	t.Log(router.String())

}

func TestRouterResource(t *testing.T) {

	router := Dispatcher{}

	expect := newExpect(t, &router)

	router.Resource(&PostsController{})
	expect("/posts", "Index")

	t.Log(router.String())

}

func TestRouterResources(t *testing.T) {

	router := Dispatcher{}

	expect := newExpect(t, &router)

	router.Resource(&InternalController{}, &ResourceOptions{Scheme: "http"})
	router.Resource(&InternalController{}, &ResourceOptions{Scheme: "https"})
	expect("/internal", "Index")

	t.Log(router.String())

}
