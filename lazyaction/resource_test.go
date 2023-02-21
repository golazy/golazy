package lazyaction

import (
	"testing"

	"golazy.dev/lazyaction/internal/router"
	"golazy.dev/lazyaction/internal/router/routertest"
)

func NewResourceTester(t *testing.T, controller interface{}, options *ResourceOptions) (func(name string, expected *router.Route), []*router.Route) {
	t.Helper()
	resource, err := newResource(controller, options)
	if err != nil {
		t.Fatal(err)
	}
	routes := resource.Routes()
	expect := func(name string, expected *router.Route) {
		t.Helper()
		t.Run(name, func(t *testing.T) {

			t.Helper()
			if expected.Verb == "" {
				expected.Verb = "GET"
			}
			err := routertest.ExpectRoute(routes, expected)
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

func TestResourceRoutes(t *testing.T) {

	controller := &PostsController{}
	expect, _ := NewResourceTester(t, controller, &ResourceOptions{})

	expect("route.ControllerName",
		&router.Route{Path: "/posts", Verb: "GET", ControllerName: "PostsController"})

	expect("route.Controller",
		&router.Route{Path: "/posts", Verb: "GET", Controller: controller})

	expect("route.Name",
		&router.Route{Path: "/posts", Verb: "GET", Name: "posts#index"})

	expect("route.Args",
		&router.Route{Path: "/posts", Verb: "GET", Args: []string{
			"*lazyaction.PostsController",
			"lazyaction.ResponseWriter",
			"*lazyaction.Request",
		}})

	expect("Index", &router.Route{Path: "/posts", Verb: "GET", Name: "posts#index"})
	expect("New", &router.Route{Path: "/posts/new", Verb: "GET", Name: "posts#new"})
	expect("VerbMethod", &router.Route{Path: "/posts/create_super", Verb: "POST", Name: "posts#create_super"})

	expect("PUT and PATCH", &router.Route{Path: "/posts/:post_id", Verb: "PUT", Name: "posts#update"})
	expect("PUT and PATCH", &router.Route{Path: "/posts/:post_id", Verb: "PATCH", Name: "posts#update"})

	expect("Member", &router.Route{Path: "/posts/:post_id/activate_later", Verb: "PUT", Name: "posts#activate_later"})

	expect("Plain action", &router.Route{Path: "/posts/about", Verb: "GET", Name: "posts#about"})
}

func TestResourceRoutes_PathNames(t *testing.T) {
	controller := &PostsController{}
	expect, _ := NewResourceTester(t, controller, &ResourceOptions{PathNames: struct{ New, Edit string }{"nuevo", "editar"}})

	expect("New route", &router.Route{Path: "/posts/nuevo", Verb: "GET", Name: "posts#new"})
	expect("Edit route", &router.Route{Path: "/posts/:post_id/editar", Verb: "GET", Name: "posts#edit"})
}

func TestResourceRoutes_Path(t *testing.T) {
	controller := &PostsController{}
	expect, _ := NewResourceTester(t, controller, &ResourceOptions{Path: "/articles"})

	expect("About route", &router.Route{Path: "/articles/about", Verb: "GET", Name: "posts#about"})
	expect("Edit route", &router.Route{Path: "/articles/:post_id/edit", Verb: "GET", Name: "posts#edit"})
}

func TestResourceRoutes_Path_Root(t *testing.T) {
	controller := &PostsController{}
	expect, _ := NewResourceTester(t, controller, &ResourceOptions{Path: "/"})

	expect("About route", &router.Route{Path: "/about", Verb: "GET", Name: "posts#about"})
	expect("Edit route", &router.Route{Path: "/:post_id/edit", Verb: "GET", Name: "posts#edit"})
}

func TestResourceRoutes_Names(t *testing.T) {
	controller := &PostsController{}
	expect, _ := NewResourceTester(t, controller, &ResourceOptions{Name: "Articles"})

	expect("Index route", &router.Route{
		Path: "/articles/:article_id", Verb: "GET",
		Name: "articles#show", Singular: "article", Plural: "articles", ParamName: ":article_id"})
}

func TestResourceRoutes_Plural(t *testing.T) {
	controller := &PostsController{}
	expect, _ := NewResourceTester(t, controller, &ResourceOptions{Plural: "articles"})

	expect("Index route", &router.Route{
		Path: "/articles/:post_id", Verb: "GET",
		Name: "posts#show", Singular: "post", Plural: "articles", ParamName: ":post_id"})
}
func TestResourceRoutes_Singular(t *testing.T) {
	controller := &PostsController{}
	expect, _ := NewResourceTester(t, controller, &ResourceOptions{Singular: "article"})

	expect("Index route", &router.Route{
		Path: "/posts/:article_id", Verb: "GET",
		Name: "posts#show", Singular: "article", Plural: "posts", ParamName: ":article_id"})

}

/*
func testResourceExpectations(t *testing.T, r *lazyaction.Resource, expectations []string) {
	t.Helper()
	routes := lazyaction.NewResource(r).ResourceActions
	if len(expectations) != len(routes) {
		t.Errorf("Expected %d routes, got %d", len(expectations), len(routes))
	}

NextRoute:
	for _, expectation := range expectations {
		for i, route := range routes {
			if expectation == route.String() {
				// Remove the route from the output
				routes = append(routes[:i], routes[i+1:]...)

				continue NextRoute
			}
		}
		t.Errorf("\nExpected: %q but was not pressent", expectation)
	}

	for _, route := range routes {
		t.Error("Got unexpected route: ", route.String())
	}
}

func TestResourceRoutes_Basic(t *testing.T) {

	testResourceExpectations(
		t,
		&lazyaction.Resource{Controller: new(PostsController)},
		[]string{
			"posts POST /posts PostsController#Create",
			"post DELETE /posts/:post_id PostsController#Destroy",
			"edit_post GET /posts/:post_id/edit PostsController#Edit",
			"posts GET /posts PostsController#Index",
			"new_post GET /posts/new PostsController#New",
			"activate_later_post PUT /posts/:post_id/activate_later PostsController#MemberPutActivateLater",
			"create_super_post POST /posts/create_super PostsController#PostCreateSuper",
			"post GET /posts/:post_id PostsController#Show",
			"post PUT|PATCH /posts/:post_id PostsController#Update",
		},
	)
}



func TestResourceRoutes_Path_NameAndSingular(t *testing.T) {

	testResourceExpectations(
		t,
		&lazyaction.Resource{Controller: new(PostsController), Path: "/", Plural: "people", Singular: "person"},
		[]string{
			"people POST / PostsController#Create",
			"person DELETE /:person_id PostsController#Destroy",
			"edit_person GET /:person_id/edit PostsController#Edit",
			"people GET / PostsController#Index",
			"new_person GET /new PostsController#New",
			"activate_later_person PUT /:person_id/activate_later PostsController#MemberPutActivateLater",
			"create_super_person POST /create_super PostsController#PostCreateSuper",
			"person GET /:person_id PostsController#Show",
			"person PUT|PATCH /:person_id PostsController#Update",
		},
	)
}

func TestResourceRoutes_ParamName(t *testing.T) {
	testResourceExpectations(
		t,
		&lazyaction.Resource{Controller: new(PostsController), ParamName: "article_id"},
		[]string{
			"posts POST /posts PostsController#Create",
			"post DELETE /posts/:article_id PostsController#Destroy",
			"edit_post GET /posts/:article_id/edit PostsController#Edit",
			"posts GET /posts PostsController#Index",
			"new_post GET /posts/new PostsController#New",
			"activate_later_post PUT /posts/:article_id/activate_later PostsController#MemberPutActivateLater",
			"create_super_post POST /posts/create_super PostsController#PostCreateSuper",
			"post GET /posts/:article_id PostsController#Show",
			"post PUT|PATCH /posts/:article_id PostsController#Update",
		},
	)
}

func TestResource_RestController(t *testing.T) {
	testResourceExpectations(
		t,
		&lazyaction.Resource{Controller: new(CommentsController)},
		[]string{
			"comments POST /comments CommentsController#Create",
			"comment DELETE /comments/:comment_id CommentsController#Destroy",
			"edit_comment GET /comments/:comment_id/edit CommentsController#Edit",
			"comments GET /comments CommentsController#Index",
			"new_comment GET /comments/new CommentsController#New",
			"comment GET /comments/:comment_id CommentsController#Show",
			"comment PUT|PATCH /comments/:comment_id CommentsController#Update",
		},
	)

}

func TestResourceRoutes_PackageController(t *testing.T) {
	testResourceExpectations(
		t,
		&lazyaction.Resource{
			Controller: new(InternalController),
		},
		[]string{
			"internal GET /internal InternalController#Index",
		},
	)

}

func TestResourceRoutes_SubResource(t *testing.T) {
	testResourceExpectations(
		t,
		&lazyaction.Resource{
			Controller: new(PostsController),
			SubResources: []*lazyaction.Resource{
				{
					Controller: new(CommentsController),
				},
			},
		},
		[]string{
			"posts POST /posts PostsController#Create",
			"post DELETE /posts/:post_id PostsController#Destroy",
			"edit_post GET /posts/:post_id/edit PostsController#Edit",
			"posts GET /posts PostsController#Index",
			"new_post GET /posts/new PostsController#New",
			"activate_later_post PUT /posts/:post_id/activate_later PostsController#MemberPutActivateLater",
			"create_super_post POST /posts/create_super PostsController#PostCreateSuper",
			"post GET /posts/:post_id PostsController#Show",
			"post PUT|PATCH /posts/:post_id PostsController#Update",

			"post_comments POST /posts/:post_id/comments CommentsController#Create",
			"post_comment DELETE /posts/:post_id/comments/:comment_id CommentsController#Destroy",
			"edit_post_comment GET /posts/:post_id/comments/:comment_id/edit CommentsController#Edit",
			"post_comments GET /posts/:post_id/comments CommentsController#Index",
			"new_post_comment GET /posts/:post_id/comments/new CommentsController#New",
			"post_comment GET /posts/:post_id/comments/:comment_id CommentsController#Show",
			"post_comment PUT|PATCH /posts/:post_id/comments/:comment_id CommentsController#Update",
		},
	)
}

*/
