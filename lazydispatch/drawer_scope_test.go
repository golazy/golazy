package lazydispatch

import "testing"

func TestScope(t *testing.T) {

	expect := func(s *Scope, method, path, name, namespace string) {
		t.Helper()
		smethod, spath, sname, snamespace, _ := s.routeInfo()
		if smethod != method {
			t.Errorf("expected method %q, got %q", method, smethod)
		}
		if spath != path {
			t.Errorf("expected path %q, got %q", path, spath)
		}
		if sname != name {
			t.Errorf("expected name %q, got %q", name, sname)
		}
		if snamespace != namespace {
			t.Errorf("expected namespace %q, got %q", namespace, snamespace)
		}
	}

	expect(&Scope{}, "GET", "/", "", "")
	expect(&Scope{path: "/"}, "GET", "/", "", "")

	parent := &Scope{method: "PUT", path: "patatas", namespace: "admin", as: "secret"}

	expect(parent, "PUT", "/patatas", "secret", "admin")
	expect(&Scope{parent: parent}, "PUT", "/patatas", "secret", "admin")
	expect(&Scope{parent: parent, method: "PUT", path: "fritas", as: "weapon", namespace: "super"}, "PUT", "/patatas/fritas", "secret_weapon", "admin/super")

}
