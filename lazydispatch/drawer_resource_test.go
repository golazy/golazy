package lazydispatch

import (
	"testing"
)

type SessionController struct {
}

func (s *SessionController) New() string {
	return "new"
}
func (s *SessionController) Delete() string {
	return "delete"
}
func (s *SessionController) Show() string {
	return "show"
}
func (s *SessionController) Create() string {
	return "create"
}
func (s *SessionController) Edit() string {
	return "edit"
}
func (s *SessionController) Update() string {
	return "update"
}
func (s *SessionController) POSTSearch() string {
	return "search"
}

func TestDrawer_Resource(t *testing.T) {
	d := newScope()

	d.Resource(&SessionController{})
	test := func(out, method, url, name, action, target string) {
		t.Helper()
		expect(t, d.routes(), out, method, url, name, action, target)
	}
	for _, r := range d.routes() {
		t.Logf("%+v", r)
	}
	test("show", "GET", "/session", "session", "show", "SessionController#Show")
	test("new", "GET", "/session/new", "new_session", "new", "SessionController#New")
	test("create", "POST", "/session", "session", "create", "SessionController#Create")
	test("edit", "GET", "/session/edit", "edit_session", "edit", "SessionController#Edit")
	test("update", "PUT,PATCH", "/session", "session", "update", "SessionController#Update")
	test("delete", "DELETE", "/session", "session", "delete", "SessionController#Delete")
	test("search", "POST", "/session/search", "session_search", "search", "SessionController#POSTSearch")
}

type ProfileController struct {
}

func TestDrawer_ResourceNested(t *testing.T) {
	d := newScope()

	d.Resource(&ProfileController{}).Draw(func(d *Scope) {
		d.Resource(&SessionController{})
	})
	test := func(out, method, url, name, action, target string) {
		t.Helper()
		expect(t, d.routes(), out, method, url, name, action, target)
	}
	for _, r := range d.routes() {
		t.Logf("%+v", r)
	}
	test("show", "GET", "/profile/session", "profile_session", "show", "SessionController#Show")
	test("new", "GET", "/profile/session/new", "new_profile_session", "new", "SessionController#New")
	test("create", "POST", "/profile/session", "profile_session", "create", "SessionController#Create")
	test("edit", "GET", "/profile/session/edit", "edit_profile_session", "edit", "SessionController#Edit")
	test("update", "PUT,PATCH", "/profile/session", "profile_session", "update", "SessionController#Update")
	test("delete", "DELETE", "/profile/session", "profile_session", "delete", "SessionController#Delete")
	test("search", "POST", "/profile/session/search", "profile_session_search", "search", "SessionController#POSTSearch")
}

func TestDrawer_ResourceScoped(t *testing.T) {
	d := newScope()

	d.Path("cool").As("admin").Namespace("private").Resource(&SessionController{})
	test := func(out, method, url, name, action, target string) {
		t.Helper()
		expect(t, d.routes(), out, method, url, name, action, target)
	}
	for _, r := range d.routes() {
		t.Logf("%+v", r)
	}
	test("show", "GET", "/cool/session", "admin_session", "show", "SessionController#Show")
	test("new", "GET", "/cool/session/new", "new_admin_session", "new", "SessionController#New")
	test("create", "POST", "/cool/session", "admin_session", "create", "SessionController#Create")
	test("edit", "GET", "/cool/session/edit", "edit_admin_session", "edit", "SessionController#Edit")
	test("update", "PUT,PATCH", "/cool/session", "admin_session", "update", "SessionController#Update")
	test("delete", "DELETE", "/cool/session", "admin_session", "delete", "SessionController#Delete")
	test("search", "POST", "/cool/session/search", "admin_session_search", "search", "SessionController#POSTSearch")

}
