package lazyaction

import (
	"net/http/httptest"
	"testing"
)

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
