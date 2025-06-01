package lazydispatch

import (
	"net/http/httptest"
	"reflect"
	"testing"
)

type Blog struct {
}

type BlogsController struct {
}

type PostsController struct {
}

func (p *PostsController) Index() string {
	return "index"
}
func (p *PostsController) Show() string {
	return "show"
}
func (p *PostsController) New() string {
	return "new"
}
func (p *PostsController) Create() string {
	return "create"
}
func (p *PostsController) Edit() string {
	return "edit"
}
func (p *PostsController) Update() string {
	return "update"
}
func (p *PostsController) Delete() string {
	return "delete"
}
func (p *PostsController) Search() string {
	return "search"
}
func (p *PostsController) POSTSearch() string {
	return "post_search"
}
func (p *PostsController) MemberGETApprove() string {
	return "member_approve"
}
func (p *PostsController) MemberPUTReject() string {
	return "member_put_reject"
}

func TestResourcesActionScope(t *testing.T) {

	tests := []struct {
		action, actionName, method, path, as, verb, namespace string
	}{
		{"New", "new", "GET", "posts/new", "post", "new", "admin"},
		{"Edit", "edit", "GET", "posts/:post_id/edit", "post", "edit", "admin"},
		{"Index", "index", "GET", "posts", "posts", "", "admin"},
		{"Show", "show", "GET", "posts/:post_id", "post", "", "admin"},
		{"Create", "create", "POST", "posts", "posts", "", "admin"},
		{"Update", "update", "PUT,PATCH", "posts/:post_id", "post", "", "admin"},
		{"Delete", "delete", "DELETE", "posts/:post_id", "post", "", "admin"},
		{"GetSearch", "search", "GET", "posts/search", "posts", "search", "admin"},
		{"Get_Search", "search", "GET", "posts/search", "posts", "search", "admin"},
		{"GETSearch", "search", "GET", "posts/search", "posts", "search", "admin"},
		{"GETSearch", "search", "GET", "posts/search", "posts", "search", "admin"},
		{"MemberGETApprove", "approve", "GET", "posts/:post_id/approve", "post", "approve", "admin"},
		{"MemberGet_Approve", "approve", "GET", "posts/:post_id/approve", "post", "approve", "admin"},
		{"MemberGETApprove", "approve", "GET", "posts/:post_id/approve", "post", "approve", "admin"},
		{"MemberGet_Approve", "approve", "GET", "posts/:post_id/approve", "post", "approve", "admin"},
		{"MemberGETApprove", "approve", "GET", "posts/:post_id/approve", "post", "approve", "admin"},
		{"MemberGetApprove", "approve", "GET", "posts/:post_id/approve", "post", "approve", "admin"},
	}

	s := newScope()
	s.path = "posts"

	r := Resources{
		scope:        s,
		plural:       "posts",
		singular:     "post",
		paramName:    "post_id",
		namespace:    "admin",
		newPathName:  "new",
		editPathName: "edit",
	}
	for _, test := range tests {
		t.Run(test.action, func(t *testing.T) {
			action, actionName, method, path, as, verb, namespace := test.action, test.actionName, test.method, test.path, test.as, test.verb, test.namespace

			t.Helper()
			s, action := r.actionScope(action)
			if action != actionName {
				t.Errorf("Expected action to be %q. Got: %q", actionName, action)
			}
			if s.method != method {
				t.Errorf("Expected method to be %q. Got: %q", method, s.method)
			}
			if s.path != path {
				t.Errorf("Expected path to be %q. Got: %q", path, s.path)
			}
			if s.as != as {
				t.Errorf("Expected as to be %q. Got: %q", as, s.as)
			}
			if s.verb != verb {
				t.Errorf("Expected verb to be %q. Got: %q", verb, s.verb)
			}
			if s.namespace != namespace {
				t.Errorf("Expected namespace to be %q. Got: %q", namespace, s.namespace)
			}
		})
	}

}

func TestDrawer_Resources(t *testing.T) {
	tests := []struct {
		out, method, url, name, action, target string
	}{
		{"index", "GET", "/posts", "posts", "index", "PostsController#Index"},
		{"show", "GET", "/posts/:post_id", "post", "show", "PostsController#Show"},
		{"new", "GET", "/posts/new", "new_post", "new", "PostsController#New"},
		{"create", "POST", "/posts", "posts", "create", "PostsController#Create"},
		{"edit", "GET", "/posts/:post_id/edit", "edit_post", "edit", "PostsController#Edit"},
		{"update", "PUT,PATCH", "/posts/:post_id", "post", "update", "PostsController#Update"},
		{"delete", "DELETE", "/posts/:post_id", "post", "delete", "PostsController#Delete"},
		{"post_search", "POST", "/posts/search", "search_posts", "search", "PostsController#POSTSearch"},
		{"member_approve", "GET", "/posts/:post_id/approve", "approve_post", "approve", "PostsController#MemberGETApprove"},
		{"member_put_reject", "PUT,PATCH", "/posts/:post_id/reject", "reject_post", "reject", "PostsController#MemberPUTReject"},
	}
	d := newScope()

	d.Resources(&PostsController{}).Model(&Post{})

	routes := d.routes()
	for _, r := range routes {
		t.Logf("%+v", r)
	}

	for _, test := range tests {
		t.Run(test.out, func(t *testing.T) {
			out, method, url, name, action, target := test.out, test.method, test.url, test.name, test.action, test.target
			expect(t, routes, out, method, url, name, action, target)
		})
	}

}

func TestDrawer_ResourcesAs(t *testing.T) {
	tests := []struct {
		out, method, url, name, action, target string
	}{
		{"index", "GET", "/posts", "articles", "index", "PostsController#Index"},
		{"show", "GET", "/posts/:post_id", "article", "show", "PostsController#Show"},
		{"new", "GET", "/posts/new", "new_article", "new", "PostsController#New"},
		{"create", "POST", "/posts", "articles", "create", "PostsController#Create"},
		{"edit", "GET", "/posts/:post_id/edit", "edit_article", "edit", "PostsController#Edit"},
		{"update", "PUT,PATCH", "/posts/:post_id", "article", "update", "PostsController#Update"},
		{"delete", "DELETE", "/posts/:post_id", "article", "delete", "PostsController#Delete"},
		{"post_search", "POST", "/posts/search", "search_articles", "search", "PostsController#POSTSearch"},
		{"member_approve", "GET", "/posts/:post_id/approve", "approve_article", "approve", "PostsController#MemberGETApprove"},
		{"member_put_reject", "PUT,PATCH", "/posts/:post_id/reject", "reject_article", "reject", "PostsController#MemberPUTReject"},
	}
	d := newScope()

	d.Resources(&PostsController{}).As("articles").Draw(func(d *Scope) {
		d.Resources(&Ideas{})
	})

	routes := d.routes()
	for _, r := range routes {
		t.Logf("%+v", r)
	}

	for _, test := range tests {
		t.Run(test.out, func(t *testing.T) {
			out, method, url, name, action, target := test.out, test.method, test.url, test.name, test.action, test.target
			expect(t, routes, out, method, url, name, action, target)
		})
	}

}
func TestDrawer_ResourcesPath(t *testing.T) {
	tests := []struct {
		out, method, url, name, action, target string
	}{
		{"index", "GET", "/articles", "posts", "index", "PostsController#Index"},
		{"show", "GET", "/articles/:post_id", "post", "show", "PostsController#Show"},
		{"new", "GET", "/articles/new", "new_post", "new", "PostsController#New"},
		{"create", "POST", "/articles", "posts", "create", "PostsController#Create"},
		{"edit", "GET", "/articles/:post_id/edit", "edit_post", "edit", "PostsController#Edit"},
		{"update", "PUT,PATCH", "/articles/:post_id", "post", "update", "PostsController#Update"},
		{"delete", "DELETE", "/articles/:post_id", "post", "delete", "PostsController#Delete"},
		{"post_search", "POST", "/articles/search", "search_posts", "search", "PostsController#POSTSearch"},
		{"member_approve", "GET", "/articles/:post_id/approve", "approve_post", "approve", "PostsController#MemberGETApprove"},
		{"member_put_reject", "PUT,PATCH", "/articles/:post_id/reject", "reject_post", "reject", "PostsController#MemberPUTReject"},
	}

	d := newScope()

	r := d.Resources(&PostsController{})
	r = r.Path("articles")
	for _, test := range tests {
		t.Run(test.out, func(t *testing.T) {
			out, method, url, name, action, target := test.out, test.method, test.url, test.name, test.action, test.target
			expect(t, d.routes(), out, method, url, name, action, target)
		})
	}

	//test := func(out, method, url, name, action, target string) {
	test := func(_, _, _, _, _, _ string) {
		t.Helper()
	}

	r.Draw(func(d *Scope) {
		d.Resources(&Ideas{})
	})

	test("ideas", "GET", "/articles/:post_id/ideas", "post_ideas", "index", "Ideas#Index")

}

func TestDrawer_Resources_RootPath(t *testing.T) {
	tests := []struct {
		out, method, url, name, action, target string
	}{
		{"index", "GET", "/", "posts", "index", "PostsController#Index"},
		{"show", "GET", "/:post_id", "post", "show", "PostsController#Show"},
		{"new", "GET", "/new", "new_post", "new", "PostsController#New"},
		{"create", "POST", "/", "posts", "create", "PostsController#Create"},
		{"edit", "GET", "/:post_id/edit", "edit_post", "edit", "PostsController#Edit"},
		{"update", "PUT,PATCH", "/:post_id", "post", "update", "PostsController#Update"},
		{"delete", "DELETE", "/:post_id", "post", "delete", "PostsController#Delete"},
		{"post_search", "POST", "/search", "search_posts", "search", "PostsController#POSTSearch"},
		{"member_approve", "GET", "/:post_id/approve", "approve_post", "approve", "PostsController#MemberGETApprove"},
		{"member_put_reject", "PUT,PATCH", "/:post_id/reject", "reject_post", "reject", "PostsController#MemberPUTReject"},
	}

	d := newScope()

	d.Resources(&PostsController{}).Path("/")
	for _, test := range tests {
		t.Run(test.out, func(t *testing.T) {
			out, method, url, name, action, target := test.out, test.method, test.url, test.name, test.action, test.target
			expect(t, d.routes(), out, method, url, name, action, target)
		})
	}

}

type Ideas struct {
}

func (i *Ideas) Index() string {
	return "ideas"
}

var ANY = struct{}{}

func TestResources_ModelsNested(t *testing.T) {
	tests := []struct {
		out    string
		models []any
	}{
		{"index", []any{ANY}},
		{"show", []any{&Blog{}, &Post{}}},
		{"new", []any{ANY}},
		{"create", []any{ANY}},
		{"edit", []any{ANY, ANY}},
		{"update", []any{&Blog{}, &Post{}}},
		{"delete", []any{&Blog{}, &Post{}}},
		{"post_search", []any{ANY}},
		{"member_approve", []any{ANY, ANY}},
		{"member_put_reject", []any{ANY, ANY}},
	}
	d := newScope()
	d.Resources(&BlogsController{}, &Blog{}).Draw(func(d *Scope) {
		d.Resources(&PostsController{}, &Post{})
	})
	routes := d.routes()

	for _, test := range tests {
		t.Run(test.out, func(t *testing.T) {
			withRoute(routes, test.out, func(r *Route) {
				t.Log(r)
				compareModels(t, r.Models, test.models)
			})
		})
	}

}

func TestResources_Models(t *testing.T) {

	tests := []struct {
		out    string
		models []any
	}{
		{"index", []any{}},
		{"show", []any{&Post{}}},
		{"new", []any{}},
		{"create", []any{}},
		{"edit", []any{struct{}{}}},
		{"update", []any{&Post{}}},
		{"delete", []any{&Post{}}},
		{"post_search", []any{}},
		{"member_approve", []any{struct{}{}}},
		{"member_put_reject", []any{struct{}{}}},
	}
	d := newScope()
	d.Resources(&PostsController{}, &Post{})
	routes := d.routes()

	for _, test := range tests {
		t.Run(test.out, func(t *testing.T) {
			withRoute(routes, test.out, func(r *Route) {
				t.Log(r)
				compareModels(t, r.Models, test.models)

			})
		})
	}

}

func compareModels(t *testing.T, models, expected []any) {
	t.Helper()
	if len(models) != len(expected) {
		t.Errorf("Expected models to be %v. Got: %v", modelsName(expected), modelsName(models))
		return
	}
	for i, m := range models {
		if reflect.TypeOf(m).String() != reflect.TypeOf(expected[i]).String() {
			t.Errorf("Expected Models to be %v. Got: %v", modelsName(expected), modelsName(models))
		}
	}

}

func modelsName(models []any) []string {
	out := []string{}
	for _, m := range models {
		out = append(out, reflect.TypeOf(m).String())
	}
	return out
}

func TestDrawer_NestedResources(t *testing.T) {
	d := newScope()

	d.Path("cool").As("admin").Namespace("private").Resources(&PostsController{})

	test := func(out, method, url, name, action, target string) {
		t.Helper()
		expect(t, d.routes(), out, method, url, name, action, target)
	}
	test("index", "GET", "/cool/posts", "admin_posts", "index", "PostsController#Index")
	test("show", "GET", "/cool/posts/:post_id", "admin_post", "show", "PostsController#Show")
	test("new", "GET", "/cool/posts/new", "new_admin_post", "new", "PostsController#New")
	test("create", "POST", "/cool/posts", "admin_posts", "create", "PostsController#Create")
	test("edit", "GET", "/cool/posts/:post_id/edit", "edit_admin_post", "edit", "PostsController#Edit")
	test("update", "PUT,PATCH", "/cool/posts/:post_id", "admin_post", "update", "PostsController#Update")
	test("delete", "DELETE", "/cool/posts/:post_id", "admin_post", "delete", "PostsController#Delete")
	test("post_search", "POST", "/cool/posts/search", "search_admin_posts", "search", "PostsController#POSTSearch")
	test("member_approve", "GET", "/cool/posts/:post_id/approve", "approve_admin_post", "approve", "PostsController#MemberGETApprove")
	test("member_put_reject", "PUT,PATCH", "/cool/posts/:post_id/reject", "reject_admin_post", "reject", "PostsController#MemberPUTReject")
}

func TestDrawer_NestedResources2(t *testing.T) {
	tests := []struct {
		out, method, path, name, action, target string
	}{
		{"index", "GET", "/blogs/:blog_id/posts", "blog_posts", "index", "PostsController#Index"},
		{"show", "GET", "/blogs/:blog_id/posts/:post_id", "blog_post", "show", "PostsController#Show"},
		{"new", "GET", "/blogs/:blog_id/posts/new", "new_blog_post", "new", "PostsController#New"},
		{"create", "POST", "/blogs/:blog_id/posts", "blog_posts", "create", "PostsController#Create"},
		{"edit", "GET", "/blogs/:blog_id/posts/:post_id/edit", "edit_blog_post", "edit", "PostsController#Edit"},
		{"update", "PUT,PATCH", "/blogs/:blog_id/posts/:post_id", "blog_post", "update", "PostsController#Update"},
		{"delete", "DELETE", "/blogs/:blog_id/posts/:post_id", "blog_post", "delete", "PostsController#Delete"},
		{"post_search", "POST", "/blogs/:blog_id/posts/search", "search_blog_posts", "search", "PostsController#POSTSearch"},
		{"member_approve", "GET", "/blogs/:blog_id/posts/:post_id/approve", "approve_blog_post", "approve", "PostsController#MemberGETApprove"},
		{"member_put_reject", "PUT,PATCH", "/blogs/:blog_id/posts/:post_id/reject", "reject_blog_post", "reject", "PostsController#MemberPUTReject"},
	}

	d := newScope()

	d.Resources(&BlogsController{}).Draw(func(d *Scope) {
		d.Resources(&PostsController{})
	})

	for _, test := range tests {
		t.Run(test.out, func(t *testing.T) {
			out, method, path, name, action, target := test.out, test.method, test.path, test.name, test.action, test.target
			expect(t, d.routes(), out, method, path, name, action, target)

		})
	}

}

func withRoute(routes []*Route, out string, fn func(r *Route)) {
	for _, route := range routes {
		r := httptest.NewRequest("", "/", nil)
		w := httptest.NewRecorder()
		route.Handler.ServeHTTP(w, r)
		if w.Body.String() == out {
			fn(route)
			return
		}
	}

	panic("not found")
}

func expect(t *testing.T, routes []*Route, out string, method, url, name, action, target string) {
	t.Helper()
	for _, route := range routes {
		r := httptest.NewRequest("", "/", nil)
		w := httptest.NewRecorder()
		route.Handler.ServeHTTP(w, r)
		if w.Body.String() == out {
			if route.Method != method {
				t.Errorf("Expected %q\tMethod to be %-10q. Got: %-20q", out, method, route.Method)
			}
			if route.URL != url {
				t.Errorf("Expected %q\tURL to be %-10q. Got: %-20q", out, url, route.URL)
			}
			if route.Name != name {
				t.Errorf("Expected %q\tName to be %-10q. Got: %-20q", out, name, route.Name)
			}
			if route.Action != action {
				t.Errorf("Expected %q\tAction to be %-10q. Got: %-20q", out, action, route.Action)
			}
			if route.Target != target {
				t.Errorf("Expected %q\tTarget to be %-10q. Got: %-20q", out, target, route.Target)
			}
			return
		}
	}
	t.Error("not found")
}
