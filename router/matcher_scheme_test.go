package router

import "testing"

func TestSchemeMatcher(t *testing.T) {

	sm := newSchemeMatcher[string]()

	expect := NewExpect(t, sm)

	expect("/", "")
	expect("http://", "")
	expect("https://", "")

	sm.Add(def("//"), s("root"))

	expect("/", "root")
	expect("http://", "root")
	expect("https://", "root")

	sm.Add(def("http://"), s("http"))

	expect("/", "root")
	expect("http://", "http")
	expect("https://", "root")

	sm.Add(def("https://"), s("https"))

	expect("/", "root")
	expect("http://", "http")
	expect("https://", "https")
}

func TestSchemeMatcher_All(t *testing.T) {

	sm := newSchemeMatcher[string]()

	sm.Add(def("/"), s("root"))
	sm.Add(def("http://"), s("http"))
	sm.Add(def("https://"), s("https"))

	has := NewExpectAll(t, sm)

	has("GET / => root")
	has("GET http:/// => http")
	has("GET https:/// => https")
}
