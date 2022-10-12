package lazyaction

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
)

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

type testRouteAction string

func (tra testRouteAction) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(tra))
}
func (tra testRouteAction) String() string {
	return string(tra)
}

func say(what string) RouteAction {
	return testRouteAction(what)
}

func exampleRouter() *Router {
	r := NewRouter()

	for _, route := range routes {
		r.Add(route[0], route[1], say(route[0]+" "+route[1]))
	}

	return r
}

func TestRouter(t *testing.T) {

	router := exampleRouter()

	for _, route := range routes {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(route[0], route[1], nil)

		router.ServeHTTP(w, r)
		expectation := route[0] + " " + route[1]
		got := strings.TrimSpace(w.Body.String())
		if got != expectation {
			t.Errorf("Expecting %q got %q", expectation, got)
		}
	}
}

func BenchmarkRouter(b *testing.B) {

	router := exampleRouter()

	b.ResetTimer()
	rand.Seed(time.Now().Unix())
	order := rand.Perm(len(routes))

	// run the Fib function b.N times
	for n := 0; n < b.N; n++ {
		w := httptest.NewRecorder()
		route := routes[order[n%len(order)]]
		r := httptest.NewRequest(route[0], route[1], nil)

		router.ServeHTTP(w, r)
	}
}
