package router

import "testing"

var def = NewRouteDefinition

func TestVerbMatcher(t *testing.T) {

	mm := newMethodMatcher[string]()

	expect := NewExpect(t, mm)

	expect("GET /", "")

	mm.Add(def("GET /"), s("get"))
	mm.Add(def("POST /"), s("post"))
	mm.Add(def("PUT /"), s("put"))
	mm.Add(def("DELETE /"), s("delete"))
	mm.Add(def("PATCH /"), s("patch"))
	mm.Add(def("OPTIONS /"), s("options"))

	expect("GET /", "get")
	expect("POST /", "post")
	expect("PUT /", "put")
	expect("DELETE /", "delete")
	expect("PATCH /", "patch")
	expect("OPTIONS /", "options")

}

func TestMethodMatcher_Multimethod(t *testing.T) {
	mm := newMethodMatcher[string]()

	expect := NewExpect(t, mm)

	mm.Add(def("GET,POST /"), s("getpost"))

	expect("GET /", "getpost")
	expect("POST /", "getpost")

	mm.Add(def("* /hi"), s("hi"))
	expect("GET /hi", "hi")
	expect("POST /hi", "hi")
	expect("PUT /hi", "hi")
	expect("PATCH /hi", "hi")
	expect("DELETE /hi", "hi")
	expect("OPTIONS /hi", "hi")
}

func TestMethodMatcher_All(t *testing.T) {
	mm := newMethodMatcher[string]()

	has := NewExpectAll(t, mm)

	mm.Add(def("GET /"), s("get"))
	mm.Add(def("PUT /"), s("put"))
	mm.Add(def("DELETE /"), s("delete"))

	has("GET / => get")
	has("PUT / => put")
	has("DELETE / => delete")
}
