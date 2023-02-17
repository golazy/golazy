package router

import (
	"math/rand"
	"testing"
	"time"
)

func TestRouteTable(t *testing.T) {

	rt := NewRouteTable[string]()
	examples := []struct{ route, url string }{}
	add := func(p, example string) {
		examples = append(examples, struct{ route, url string }{p, example})
		rt.Add(p, &p)
	}

	add("/", "/")
	add("/:name", "/welcome")
	add("/:name/show", "/photo/show")
	add("/:page/publish", "/about/publish")
	add("/posts/:id", "/posts/33")
	add("/users/:id", "/users/33")
	add("/users/new", "/users/new")
	add("/users/:id/censor", "/users/33/censor")

	for _, e := range examples {
		r := rt.Find(e.url)
		if r == nil {
			t.Errorf("route %q not found", e.route)
			continue
		}
		if *r != e.route {
			t.Errorf("expected Find(%q) => %q. Got: %q", e.url, e.route, *r)
		}
	}
}

func BenchmarkRouteTableFind(b *testing.B) {

	rt := NewRouteTable[string]()
	for _, r := range routes {
		if rt.Find(r[1]) != nil {
			rt.Add(r[1], &r[1])
		}
	}

	rand.Seed(time.Now().Unix())
	order := rand.Perm(10)

	// run the Fib function b.N times
	for n := 0; n < b.N; n++ {
		r := routes[order[n%len(order)]]
		rt.Find(r[1])
	}
}
