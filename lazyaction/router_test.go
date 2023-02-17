package lazyaction_test

import (
	"net/http/httptest"
	"testing"

	"golazy.dev/lazyaction"
)

func TestApp(t *testing.T) {

	a := lazyaction.Router{
		Name: "TestApp",
	}

	f1 := func(id string) (html string, err error) {
		return id, nil
	}

	a.Route("/", f1)
	response := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/hola", nil)
	a.ServeHTTP(response, request)

	t.Error(response.Body.String())

}

func TestExtractParam2(t *testing.T) {
	route := "/posts/:id"
	path := "/posts/1"
	params := lazyaction.ExtractParam2(route, path)
	if params["id"] != "1" {
		t.Error(params)
	}
}
