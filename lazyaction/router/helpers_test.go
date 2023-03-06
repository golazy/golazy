package router

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func s(s string) *string {
	return &s
}

func u(p string) *http.Request {
	method := "GET"
	if strings.Contains(p, " ") {
		parts := strings.Split(p, " ")
		method = parts[0]
		p = parts[1]
	}
	r, err := http.NewRequest(method, p, nil)
	if err != nil {
		panic(err)
	}
	return r
}

func NewExpect(t *testing.T, m Matcher[string]) func(url string, expected string) {
	t.Helper()
	return func(url string, expected string) {
		t.Helper()
		r := m.Find(u(url))
		if expected == "" {
			if r == nil {
				return
			}
			t.Errorf("url: %+v should not exists. Got: %v", url, *r)
			return
		}

		if r == nil {
			t.Errorf("url: %+v should exists", url)
			return
		}
		if *r != expected {
			t.Errorf("url: %+v should return %v. Got: %v", url, expected, *r)
		}
	}
}

func RoutesToTable(routes []Route[string]) string {
	routesS := ""
	for _, r := range routes {
		routesS += fmt.Sprintf("%v %v => %s\n", r.Req.Method, r.Req.URL.String(), *r.T)
	}
	return routesS
}

func NewExpectAll(t *testing.T, m Matcher[string]) func(expected string) {
	t.Helper()
	return func(expected string) {
		t.Helper()
		for _, route := range m.All() {
			if expected == route.String() {
				return
			}
		}
		t.Errorf("Expected route %q not found", expected)
		t.Log(m.All())
	}
}
