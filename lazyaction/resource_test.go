package lazyaction

import (
	"testing"

	"net/url"

	"golazy.dev/lazyaction/internal/args"
)

func NewResourceTester(t *testing.T, controller any, options *ResourceOptions) (func(name string, expected *Action), []*Action) {
	t.Helper()
	resource, err := newResource(controller, options)
	if err != nil {
		t.Fatal(err)
	}
	routes := resource.Actions()
	expect := func(name string, expected *Action) {
		t.Helper()
		t.Run(name, func(t *testing.T) {

			t.Helper()
			if expected.Method == "" {
				expected.Method = "GET"
			}
			err := ExpectAction(routes, expected)
			if err != nil {
				t.Error(err)
			}
		})
	}
	t.Log("routes: ")
	for _, route := range routes {
		t.Log(route.String())
	}

	return expect, routes
}

func u(p string) url.URL {
	u, err := url.Parse(p)
	if err != nil {
		panic(err)
	}
	return *u
}

func TestResourceActions(t *testing.T) {

	controller := &PostsController{}
	expect, _ := NewResourceTester(t, controller, &ResourceOptions{})

	expect("route.ControllerName",
		&Action{URL: u("/posts"), Method: "GET", ControllerName: "PostsController"})

	expect("route.Controller",
		&Action{URL: u("/posts"), Method: "GET", Controller: controller})

	expect("route.Name",
		&Action{URL: u("/posts"), Method: "GET", Name: "posts#index"})

	expect("route.Args",
		&Action{
			URL:    u("/posts"),
			Method: "GET",
			Fn:     &args.Fn{Ins: []string{"http.ResponseWriter", "*http.Request"}},
		},
	)

	expect("Index", &Action{URL: u("/posts"), Method: "GET", Name: "posts#index"})
	expect("New", &Action{URL: u("/posts/new"), Method: "GET", Name: "posts#new"})
	expect("VerbMethod", &Action{URL: u("/posts/create_super"), Method: "POST", Name: "posts#create_super"})

	expect("PUT and PATCH", &Action{URL: u("/posts/:post_id"), Method: "PUT,PATCH", Name: "posts#update"})

	expect("Member", &Action{URL: u("/posts/:post_id/activate_later"), Method: "PUT", Name: "posts#activate_later"})

	expect("Plain action", &Action{URL: u("/posts/about"), Method: "GET", Name: "posts#about"})
}

func TestResourceActions_PathNames(t *testing.T) {
	controller := &PostsController{}
	expect, _ := NewResourceTester(t, controller, &ResourceOptions{PathNames: struct{ New, Edit string }{"nuevo", "editar"}})

	expect("New route", &Action{URL: u("/posts/nuevo"), Method: "GET", Name: "posts#new"})
	expect("Edit route", &Action{URL: u("/posts/:post_id/editar"), Method: "GET", Name: "posts#edit"})
}

func TestResourceActions_Path(t *testing.T) {
	controller := &PostsController{}
	expect, _ := NewResourceTester(t, controller, &ResourceOptions{Path: "/articles"})

	expect("About route", &Action{URL: u("/articles/about"), Method: "GET", Name: "posts#about"})
	expect("Edit route", &Action{URL: u("/articles/:post_id/edit"), Method: "GET", Name: "posts#edit"})
}

func TestResourceActions_Path_Root(t *testing.T) {
	controller := &PostsController{}
	expect, _ := NewResourceTester(t, controller, &ResourceOptions{Path: "/"})

	expect("About route", &Action{URL: u("/about"), Method: "GET", Name: "posts#about"})
	expect("Edit route", &Action{URL: u("/:post_id/edit"), Method: "GET", Name: "posts#edit"})
}

func TestResourceActions_Names(t *testing.T) {
	controller := &PostsController{}
	expect, _ := NewResourceTester(t, controller, &ResourceOptions{Name: "Articles"})

	expect("Index route", &Action{
		URL: u("/articles/:article_id"), Method: "GET",
		Name: "articles#show", Singular: "article", Plural: "articles", ParamName: ":article_id"})
}

func TestResourceActions_Plural(t *testing.T) {
	controller := &PostsController{}
	expect, _ := NewResourceTester(t, controller, &ResourceOptions{Plural: "articles"})

	expect("Index route", &Action{
		URL: u("/articles/:post_id"), Method: "GET",
		Name: "posts#show", Singular: "post", Plural: "articles", ParamName: ":post_id"})
}
func TestResourceActions_Singular(t *testing.T) {
	controller := &PostsController{}
	expect, _ := NewResourceTester(t, controller, &ResourceOptions{Singular: "article"})

	expect("Index route", &Action{
		URL: u("/posts/:article_id"), Method: "GET",
		Name: "posts#show", Singular: "article", Plural: "posts", ParamName: ":article_id"})

}

func TestResourceActions_Scheme(t *testing.T) {
	controller := &PostsController{}
	expect, _ := NewResourceTester(t, controller, &ResourceOptions{Scheme: "http"})

	expect("http route", &Action{URL: u("http:///posts"), Method: "GET", Name: "posts#index"})

}

func TestResourceActions_Domain(t *testing.T) {
	controller := &PostsController{}
	expect, _ := NewResourceTester(t, controller, &ResourceOptions{Domain: "api.*"})

	expect("http route", &Action{URL: u("//api.*/posts"), Method: "GET", Name: "posts#index"})

}

func TestResourceActions_Port(t *testing.T) {
	controller := &PostsController{}
	expect, _ := NewResourceTester(t, controller, &ResourceOptions{Port: "9000"})

	expect("http route", &Action{URL: u("//:9000/posts"), Method: "GET", Name: "posts#index"})
}

func TestResourceActions_Host(t *testing.T) {
	controller := &PostsController{}
	expect, _ := NewResourceTester(t, controller, &ResourceOptions{Port: "9000", Domain: "api.*"})

	expect("http route", &Action{URL: u("//api.*:9000/posts"), Method: "GET", Name: "posts#index"})

}
