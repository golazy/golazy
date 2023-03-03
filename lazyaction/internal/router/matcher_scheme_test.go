package router

import "testing"

func TestSchemeMatcher(t *testing.T) {

	sm := NewSchemeMatcher[string]()

	expect := NewExpect(t, sm)

	expect("/", "")
	expect("http://", "")
	expect("https://", "")

	sm.Add(u("//"), s("root"))

	expect("/", "root")
	expect("http://", "root")
	expect("https://", "root")

	sm.Add(u("http://"), s("http"))

	expect("/", "root")
	expect("http://", "http")
	expect("https://", "root")

	sm.Add(u("https://"), s("https"))

	expect("/", "root")
	expect("http://", "http")
	expect("https://", "https")
}

func TestSchemeMatcher_All(t *testing.T) {

	sm := NewSchemeMatcher[string]()

	sm.Add(u("/"), s("root"))
	sm.Add(u("http://"), s("http"))
	sm.Add(u("https://"), s("https"))

	has := NewExpectAll(t, sm)

	has("GET / => root")
	has("GET http:/// => http")
	has("GET https:/// => https")
}
