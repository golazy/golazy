package lazydispatch

import (
	"fmt"
	"net/http"
	"net/url"
	"path"
	"reflect"
	"strings"
)

type Route struct {

	// URL is the patch to match
	// It supports two kind of wildcards:
	// - :post_id as a named paramenter
	// - * as a catch all that will match anything
	//
	// It is also extension aware, so the following request will match the same route:
	// - /posts/1
	// - /posts/1.html // Routes to /posts/1 and sets the content-type to text/html
	// - /posts/1.json // Rotues to /posts/1 and sets the content-type to application/json
	// - /posts/1/      // Trailing slash are ignored
	//
	// The url can include ports, domains and schemas:
	// - http://example.com:8080/posts/1
	// - https://example.com/posts/1
	// - /example.com/posts/1
	// - //:3000/posts/1
	// - http://:3000/posts/1
	URL  string
	Path string // Path is the path component of the URL.

	Method string // One of GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD

	// Models allows the router to generate paths just by providing the model
	// for example path_for(&User{}) will return /users/:id
	// In case of a nested resource it will require the parent model
	// for example path_for(&Account{}, &Post{}) will return /accounts/:account_id/posts/:post_id
	// In case there are named routes, the model can use the name to generate the path
	// for example path_for(&User{}, "profile") will return /users/:id/profile
	// There is an explicit way to get the path:
	// path_for("posts", &Post{Id:3}) will return /posts/3
	// path_for("posts", &Post{Id:3}, "comments") will return /posts/3/comments
	// path_for("posts", &Post{Id:2}, &Comment{Id:5}, "review") will return /posts/2/comments/5/review
	// The previous path can also be expressed as:
	// path_for("review_posts_comment", &Post{}, &Comment{}) will return /posts/:post_id/comments/:comment_id/review
	Models []any

	// Name holds the name of the route
	Name string

	// Action Is the action on the resource. For example "review" on a post
	Action string

	// Controller is the name of the controller. For example "posts"
	Controller string

	// Namespace is the namespace of the controller
	Namespace string

	// Target is the name of the controller. For example "PostsController#Index" It is only use for debugging purposes
	Target string

	Handler http.Handler `json:"-"`
}

func (r *Route) String() string {
	return fmt.Sprintf("route:%s %s (name: %s) (target:%s) (action: %s)", r.Method, r.URL, r.Name, r.Target, r.Action)

}

func (r *Route) normalize() *Route {
	r.assignModels()
	r.assignDefaultMethod()
	r.prefixURL()
	r.assignPath()
	return r
}
func (route *Route) assignPath() {
	u, err := url.Parse(route.URL)
	if err != nil {
		err = fmt.Errorf("can't parse url %s %s: %+v", route.URL, route.Name, route)
		panic(err)
	}
	route.Path = u.Path
}
func (route *Route) assignDefaultMethod() {
	if route.Method == "" {
		route.Method = "GET"
	}
}
func (route *Route) prefixURL() {
	if route.URL == "" || route.URL[0] != '/' {
		route.URL = path.Join("/", route.URL)
	}
}
func (route *Route) assignModels() {
	segments := strings.Split(route.URL, "/")
	nParams := 0
	for _, s := range segments {
		if strings.HasPrefix(s, ":") || s == "*" {
			nParams++
		}
	}
	if len(route.Models) == nParams {
		return
	}
	if len(route.Models) == 0 {
		for i := 0; i < nParams; i++ {
			route.Models = append(route.Models, any(struct{}{}))
		}
		return
	}
	modelsNames := []string{}
	for _, m := range route.Models {
		modelsNames = append(modelsNames, reflect.TypeOf(m).Name())
	}

	msg := fmt.Sprintf("when providing Models to a route, the number of models should equal to the number of parameters in the path.\n"+
		"the path %q requires %d parameters, but %v where provide.", route.URL, nParams, modelsNames)
	fmt.Println(msg)
}
