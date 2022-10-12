package lazyaction

import (
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
)

func SendRequest(router Router, method, path string) (body string, res *httptest.ResponseRecorder) {
	r := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	return strings.TrimSpace(w.Body.String()), w
}

func ExampleController_Routes() {

	rt := NewRouter()
	rt.Resource(new(PostsController))

	fmt.Println(rt.String())

	// Output:
	// METHOD PATH                           DESTINATION
	// GET    /posts                         PostsController#Index
	// POST   /posts                         PostsController#Create
	// POST   /posts/create_super            PostsController#PostCreateSuper
	// GET    /posts/new                     PostsController#New
	// GET    /posts/:post_id                PostsController#Show
	// PUT    /posts/:post_id                PostsController#Update
	// PATCH  /posts/:post_id                PostsController#Update
	// DELETE /posts/:post_id                PostsController#Delete
	// PUT    /posts/:post_id/activate_later PostsController#MemberPutActivateLater
}

func TestResourceHandler(t *testing.T) {
	rt := NewRouter()
	rt.Resource(new(MultiArgsController))
	routes := rt.Table().Values
	t.Log(routes)
	route := rt.byMethod[0].find("multi_args/24")
	if route == nil {
		t.Fatal("Can't find the route")
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/multi_args/24", nil)
	(*route).ServeHTTP(w, r)

	if w.Body.String() == "hello" {
		t.Fatal(w.Body.String())
	}

}
