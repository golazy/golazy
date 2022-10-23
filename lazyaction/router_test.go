package lazyaction

import (
	"fmt"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestRouter(t *testing.T) {

	router := NewRouter()
	router.AddResourceDefinition(&ResourceDefinition{Controller: new(PostsController)})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/posts", nil)

	router.ServeHTTP(w, r)

	if w.Body.String() != "Index" {
		t.Error(w.Body.String())
	}

}

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
