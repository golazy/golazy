package router

import (
	"net/http"
	"testing"
)

type TestRoute struct {
	Name string
}

func TestRouter(t *testing.T) {
	router := NewRouter[TestRoute]()
	path := "GET /brands/:brand_id/inputs/:prompt_input_id/ideas"
	router.AddByPath(path, &TestRoute{Name: "ideas"})

	req, err := http.NewRequest("GET", "/brands/123/inputs/456/ideas", nil)
	if err != nil {
		t.Fatal(err)
	}
	r := router.Find(req)
	if r == nil || r.Name != "ideas" {
		t.Fatal("Missing route")
	}

}

/*
func TestRouter(t *testing.T) {
	router := NewRouter[TestRoute]()

	router.Add("GET", "/posts", &TestRoute{Name: "test_route"})

	if r := router.Find("GET", "/posts"); r == nil || r.Name != "test_route" {
		t.Error("Missing route", r.Name)
	}
}

func TestProtocol(t *testing.T) {
	router := NewRouter[string]()
	router.Add("GET", "http:///posts", s("http"))

	test := func(path, expected string) {
		r := router.Find("GET", path)
		if r == nil {
			t.Errorf("route %q not found", path)
			return
		}
		if *r != expected {
			t.Errorf("expected Find(%q) => %q. Got: %q", path, expected, *r)
		}
	}

	test("http://localhost/posts", "http")
	test("/posts", "http")

}

*/
