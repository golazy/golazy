package router

import "testing"

func TestPortMatcher(t *testing.T) {
	pm := NewPortMatcher[string]()
	expect := NewExpect(t, pm)

	expect("//localhost:43", "")

	pm.Add(u("//localhost:8080"), s("8080"))
	expect("//localhost:43", "")

	pm.Add(u("//localhost:8080/asdf"), s("8080"))
	pm.Add(u("//localhost"), s("0"))

	expect("//localhost:43", "0")

	expect("//localhost:8080", "8080")
	expect("//localhost", "0")

	expect("http://localhost:43", "0")

}

func TestPortMatcher_All(t *testing.T) {

	pm := NewPortMatcher[string]()
	pm.Add(u("//localhost:8080"), s("8080"))
	pm.Add(u("//localhost:8080/asdf"), s("asdf"))
	pm.Add(u("//localhost"), s("0"))

	has := NewExpectAll(t, pm)

	has("GET //:8080/ => 8080")
	has("GET //:8080/asdf => asdf")
	has("GET / => 0")

}
