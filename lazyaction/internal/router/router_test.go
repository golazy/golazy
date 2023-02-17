package router

import (
	"fmt"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
)

type TestRoute struct {
	Name string
}

func TestRouter(t *testing.T) {
	router := NewRouter[TestRoute]()

	route := &Route[TestRoute]{
		Verb:   "GET",
		Path:   "/posts",
		Name:   "test_route",
		Target: TestRoute{Name: "test_route"},
	}

	router.Add(route)

	req := httptest.NewRequest("GET", "/posts", nil)

	if r := router.Find(req); r == nil || r.Name != "test_route" {
		t.Error("Missing route", r.Name)
	}
}

/*
func TestRouter(t *testing.T) {

	router := NewRouter()
	router.AddResourceDefinition(&ResourceDefinition{Controller: new(ArticlesController)})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/posts", nil)

	router.ServeHTTP(w, r)

	if w.Body.String() != "Index" {
		t.Error(w.Body.String())
	}

}

*/

var routes [][]string

func init() {
	data, err := os.ReadFile("routes.txt")
	if err != nil {
		panic(err)
	}

	whitespaces := regexp.MustCompile(`\s+`)

	routes = make([][]string, 0, 1500)
	for _, line := range strings.Split(string(data), "\n") {
		cleanLine := whitespaces.ReplaceAllString(line, " ")
		parts := strings.Split(cleanLine, " ")
		if len(parts) != 2 {
			panic(fmt.Sprintf("%q", line))
		}
		if methodIndex(parts[0]) < 0 {
			panic(parts[0] + parts[1])
		}
		routes = append(routes, []string{parts[0], parts[1]})
	}
}
