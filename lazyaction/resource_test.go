package lazyaction

import (
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
)

func ExampleResource() {

	routes := Resource(new(PostsController))
	fmt.Println(routes)

	// Output:
	// METHOD PATH                           DESTINATION
	// POST   /posts                         PostsController#Create
	// GET    /posts                         PostsController#Index
	// POST   /posts/create_super            PostsController#PostCreateSuper
	// GET    /posts/new                     PostsController#New
	// DELETE /posts/:post_id                PostsController#Delete
	// GET    /posts/:post_id                PostsController#Show
	// PUT    /posts/:post_id                PostsController#Update
	// PATCH  /posts/:post_id                PostsController#Update
	// PUT    /posts/:post_id/activate_later PostsController#MemberPutActivateLater

}

func TestController(t *testing.T) {

	say := func(what string) Handler {
		return HandlerFunc(func(w ResponseWriter, r *Request) {
			w.Write([]byte(fmt.Sprint(what, r.Params)))
		})
	}
	c := Router{Resource(new(PostsController))}

	testRoute := func(method, path, expectation string) {
		t.Helper()
		r := httptest.NewRequest(method, path, nil)
		w := httptest.NewRecorder()
		c.ServeHTTP(w, r)
		if strings.TrimSpace(w.Body.String()) != expectation {
			t.Errorf("Expecting %q to send %q. Got %q", path, expectation, w.Body.String())
		}
	}

	testRoute("GET", "/posts", "Index")
	testRoute("PUT", "/posts/39/activate_later", "ActivateLater 39")
	testRoute("GET", "/posts", "Index")
	testRoute("GET", "/", "Not Found")
	testRoute("POST", "/posts", "Create")
	testRoute("POST", "/posts/create_super", "CreateSuper")
	testRoute("GET", "/posts/new", "New")
	testRoute("PUT", "/posts/39", "Update 39")
	testRoute("PATCH", "/posts/39", "Update 39")
	testRoute("DELETE", "/posts/39", "Delete 39")
	testRoute("GET", "/posts/", "<a href=\"/posts\">Permanent Redirect</a>.")         // Redirects trailing slashes
	testRoute("GET", "/posts/111/", "<a href=\"/posts/111\">Permanent Redirect</a>.") // Redirects trailing slashes
}
