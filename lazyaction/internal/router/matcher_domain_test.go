package router

import "testing"

func TestDomainMatcher(t *testing.T) {

	dm := NewDomainMatcher[string]()

	expect := NewExpect(t, dm)

	expect("/", "")

	dm.Add(u("//localhost"), s("localhost"))
	expect("/", "")

	dm.Add(u("//*/"), s("root"))

	expect("//", "root")

	dm.Add(u("//(api,www).google.(net,org)/"), s("complex"))
	expect("//api.google.net", "complex")
	expect("//www.google.org", "complex")

}

func TestDomainMatcher_All(t *testing.T) {
	dm := NewDomainMatcher[string]()

	dm.Add(u("//localhost"), s("localhost"))
	dm.Add(u("//*/"), s("root"))
	dm.Add(u("//(api,www).google.(net,org):443/"), s("complex"))

	has := NewExpectAll(t, dm)

	has("GET //localhost/ => localhost")
	has("GET //(api,www).google.(net,org):443/ => complex")
	has("GET / => root")
}
