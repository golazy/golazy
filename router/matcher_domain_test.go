package router

import "testing"

func TestDomainMatcher(t *testing.T) {

	dm := newDomainMatcher[string]()

	expect := NewExpect(t, dm)

	expect("/", "")

	dm.Add(def("//localhost"), s("localhost"))
	expect("/", "")

	dm.Add(def("//*/"), s("root"))

	expect("//", "root")

	dm.Add(def("//(api,www).google.(net,org)/"), s("complex"))
	expect("//api.google.net", "complex")
	expect("//www.google.org", "complex")

}

func TestDomainMatcher_All(t *testing.T) {
	dm := newDomainMatcher[string]()

	dm.Add(def("//localhost"), s("localhost"))
	dm.Add(def("//*/"), s("root"))
	dm.Add(def("//(api,www).google.(net,org):443/"), s("complex"))

	has := NewExpectAll(t, dm)

	has("GET //localhost/ => localhost")
	has("GET //(api,www).google.(net,org):443/ => complex")
	has("GET / => root")
}
