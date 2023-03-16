package lazyaction

import (
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestRouterDispatch(t *testing.T) {
	router := Dispatcher{}
	router.Route("/", StringHandler("root"))

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Body.String() != "root" {
		t.Errorf("expected root got %q", rr.Body.String())
	}
}

type C struct {
	count int
}

func (c *C) Index() string {
	c.count++
	return strconv.Itoa(c.count)
}

func TestRouterDispatch_NewInstance(t *testing.T) {
	router := Dispatcher{}
	router.Resource(&C{})

	req := httptest.NewRequest("GET", "/c", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)
	req = httptest.NewRequest("GET", "/c", nil)
	rr = httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Body.String() != "1" {
		t.Errorf("expected root got %q", rr.Body.String())
	}
}
