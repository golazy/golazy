package lazyaction

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouterRoute(t *testing.T) {

	router := Routes{}

	expect := func(path, expected string) {
		req, err := http.NewRequest("GET", path, nil)
		if err != nil {
			t.Fatal(err)
		}
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Body.String() != expected {
			t.Errorf("expected %q to return %q got %q", path, expected, rr.Body.String())
		}
	}

	router.Route("/", StringHandler("root"))
	router.Route("/hi", StringHandler("hi"))
	expect("/", "root")
	expect("/hi", "hi")

	t.Log(router.String())

}

func TestRouterResource(t *testing.T) {

	router := Routes{}

	expect := func(path, expected string) {
		req, err := http.NewRequest("GET", path, nil)
		if err != nil {
			t.Fatal(err)
		}
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Body.String() != expected {
			t.Errorf("expected %q to return %q got %q", path, expected, rr.Body.String())
		}
	}

	router.Resource(&PostsController{})
	expect("/posts", "Index")

	t.Error(router.String())

}

func TestRouterDispatch(t *testing.T) {
	router := Routes{}
	router.Route("/", StringHandler("root"))

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Body.String() != "root" {
		t.Errorf("expected root got %q", rr.Body.String())
	}
}
