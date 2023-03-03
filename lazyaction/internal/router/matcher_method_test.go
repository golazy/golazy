package router

import "testing"

func TestVerbMatcher(t *testing.T) {

	mm := NewMethodMatcher[string]()

	expect := NewExpect(t, mm)

	expect("GET /", "")

	mm.Add(u("GET /"), s("get"))
	mm.Add(u("POST /"), s("post"))
	mm.Add(u("PUT /"), s("put"))
	mm.Add(u("DELETE /"), s("delete"))
	mm.Add(u("PATCH /"), s("patch"))
	mm.Add(u("OPTIONS /"), s("options"))

	expect("GET /", "get")
	expect("POST /", "post")
	expect("PUT /", "put")
	expect("DELETE /", "delete")
	expect("PATCH /", "patch")
	expect("OPTIONS /", "options")

}

func TestMethodMatcher_All(t *testing.T) {
	mm := NewMethodMatcher[string]()

	has := NewExpectAll(t, mm)

	mm.Add(u("GET /"), s("get"))
	mm.Add(u("PUT /"), s("put"))
	mm.Add(u("DELETE /"), s("delete"))

	has("GET / => get")
	has("PUT / => put")
	has("DELETE / => delete")
}
