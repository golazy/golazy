package router

import (
	"testing"
)

func TestRouteTable(t *testing.T) {

	rt := NewPathMatcher[string]()

	expect := NewExpect(t, rt)

	rt.Add(def("/"), s("root"))
	rt.Add(def("/:name"), s("name"))
	rt.Add(def("/:name/edit"), s("name_edit"))
	rt.Add(def("/posts/:id"), s("posts_show"))
	rt.Add(def("/posts/new"), s("posts_new"))
	rt.Add(def("/users/:id/censor"), s("users_censor"))

	expect("/", "root")
	expect("/asdf", "name")
	expect("/asdf/edit", "name_edit")
	expect("/posts/123", "posts_show")
	expect("/posts/new", "posts_new")
	expect("/users/123/censor", "users_censor")
	expect("/users/npm:@hotwired%2Fturbo/censor", "users_censor")

}

func TestPathMatcher_Wildcard(t *testing.T) {

	rt := NewPathMatcher[string]()

	expect := NewExpect(t, rt)

	rt.Add(def("/api/*"), s("api"))

	expect("/api", "api")
	expect("/api/", "api")
	expect("/api/asdf", "api")
	expect("/api/asdf", "api")

}

func TestPathMatcher_All(t *testing.T) {
	rt := NewPathMatcher[string]()
	rt.Add(def("/path"), s("root"))

	expectHas := NewExpectAll(t, rt)

	expectHas("GET /path => root")

}
