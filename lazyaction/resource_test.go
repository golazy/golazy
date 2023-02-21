package lazyaction

import (
	"errors"
	"fmt"
	"testing"

	"golazy.dev/lazyaction/internal/router"
)

func TestResourceRoutes(t *testing.T) {

	controller := &PostsController{}
	resource, err := newResource(controller, &ResourceOptions{})
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
			r := findRoute(routes, expected)
			if r == nil {
				t.Error("Route not found: " + expected.Verb + " " + expected.Path)
				return
			}
			if err := compareRoute(r, expected); err != nil {
				t.Error(err, fmt.Sprintf("%+v", *r))
			}
		})
	}

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

	for _, route := range routes {
		t.Log(route.String())
	}

}

func compareRoute(original, expected *router.Route) error {
	var errs []error
	if original == nil || expected == nil {
		return fmt.Errorf("Missing routes to compare")
	}

	if expected.Verb != "" {
		if original.Verb != expected.Verb {
			errs = append(errs, fmt.Errorf("Expected verb %s, got %s", expected.Verb, original.Verb))
		}
	} else {
		if original.Verb != "GET" {
			errs = append(errs, fmt.Errorf("Expected empty verb to generate GET, got %s", original.Verb))
		}
	}

	if expected.Path != "" {
		if original.Path != expected.Path {
			errs = append(errs, fmt.Errorf("Expected path %s, got %s", expected.Path, original.Path))
		}
	}

	if expected.Name != "" {
		if original.Name != expected.Name {
			errs = append(errs, fmt.Errorf("Expected name %s, got %s", expected.Name, original.Name))
		}
	}

	if expected.Target != nil {
		if original.Target != expected.Target {
			errs = append(errs, fmt.Errorf("Expected target %s, got %s", expected.Target, original.Target))
		}
	}

	if expected.Args != nil {
		if len(original.Args) != len(expected.Args) {
			errs = append(errs, fmt.Errorf("Expected %v arguments, got %v", expected.Args, original.Args))
		}

		for i, arg := range expected.Args {
			if original.Args[i] != arg {
				errs = append(errs, fmt.Errorf("Expected argument %s, got %s", arg, original.Args[i]))
			}
		}
	}

	if expected.Rets != nil {
		if len(original.Rets) != len(expected.Rets) {
			errs = append(errs, fmt.Errorf("Expected %d return values, got %d", len(expected.Rets), len(original.Rets)))
		}

		for i, ret := range expected.Rets {
			if original.Rets[i] != ret {
				errs = append(errs, fmt.Errorf("Expected return value %s, got %s", ret, original.Rets[i]))
			}
		}
	}

	if expected.Controller != nil {
		if original.Controller != expected.Controller {
			errs = append(errs, fmt.Errorf("Expected controller %s, got %s", expected.Controller, original.Controller))
		}
	}

	if expected.ControllerName != "" {
		if original.ControllerName != expected.ControllerName {
			errs = append(errs, fmt.Errorf("Expected controller name %s, got %s", expected.ControllerName, original.ControllerName))
		}
	}

	if expected.Plural != "" {
		if original.Plural != expected.Plural {
			errs = append(errs, fmt.Errorf("Expected plural %s, got %s", expected.Plural, original.Plural))
		}
	}

	if expected.Singular != "" {
		if original.Singular != expected.Singular {
			errs = append(errs, fmt.Errorf("Expected singular %s, got %s", expected.Singular, original.Singular))
		}
	}

	if expected.ParamName != "" {
		if original.ParamName != expected.ParamName {
			errs = append(errs, fmt.Errorf("Expected param name %s, got %s", expected.ParamName, original.ParamName))
		}
	}

	return errors.Join(errs...)
}

func findRoute(routes []*router.Route, expected *router.Route) *router.Route {
	for _, route := range routes {
		if route.Path == expected.Path && route.Verb == expected.Verb {
			return route
		}
	}
	return nil
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

func TestResourceRoutes_PathNames(t *testing.T) {

	testResourceExpectations(
		t,
		&lazyaction.Resource{Controller: new(PostsController), PathNames: struct{ New, Edit string }{"nuevo", "editar"}},
		[]string{
			"posts POST /posts PostsController#Create",
			"post DELETE /posts/:post_id PostsController#Destroy",
			"edit_post GET /posts/:post_id/editar PostsController#Edit",
			"posts GET /posts PostsController#Index",
			"new_post GET /posts/nuevo PostsController#New",
			"activate_later_post PUT /posts/:post_id/activate_later PostsController#MemberPutActivateLater",
			"create_super_post POST /posts/create_super PostsController#PostCreateSuper",
			"post GET /posts/:post_id PostsController#Show",
			"post PUT|PATCH /posts/:post_id PostsController#Update",
		},
	)
}

func TestResourceRoutes_Path(t *testing.T) {

	testResourceExpectations(
		t,
		&lazyaction.Resource{Controller: new(PostsController), Path: "/articles"},
		[]string{
			"posts POST /articles PostsController#Create",
			"post DELETE /articles/:post_id PostsController#Destroy",
			"edit_post GET /articles/:post_id/edit PostsController#Edit",
			"posts GET /articles PostsController#Index",
			"new_post GET /articles/new PostsController#New",
			"activate_later_post PUT /articles/:post_id/activate_later PostsController#MemberPutActivateLater",
			"create_super_post POST /articles/create_super PostsController#PostCreateSuper",
			"post GET /articles/:post_id PostsController#Show",
			"post PUT|PATCH /articles/:post_id PostsController#Update",
		},
	)
}

func TestResourceRoutes_Path_TopLevel(t *testing.T) {

	testResourceExpectations(
		t,
		&lazyaction.Resource{Controller: new(PostsController), Path: "/"},
		[]string{
			"posts POST / PostsController#Create",
			"post DELETE /:post_id PostsController#Destroy",
			"edit_post GET /:post_id/edit PostsController#Edit",
			"posts GET / PostsController#Index",
			"new_post GET /new PostsController#New",
			"activate_later_post PUT /:post_id/activate_later PostsController#MemberPutActivateLater",
			"create_super_post POST /create_super PostsController#PostCreateSuper",
			"post GET /:post_id PostsController#Show",
			"post PUT|PATCH /:post_id PostsController#Update",
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
