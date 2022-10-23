package lazyaction

import (
	"testing"
	"time"
)

type Comment struct {
	Comment   string
	Author    string
	CreatedAt time.Time
}

type CommentsController struct {
	RestController[Comment]
}

type PostsController struct {
}

func (p *PostsController) Index(w ResponseWriter, r *Request) {
	w.Write([]byte("Index"))
}

func (p *PostsController) New(w ResponseWriter, r *Request) {
	w.Write([]byte("New"))
}
func (p *PostsController) Edit(w ResponseWriter, r *Request) {
	w.Write([]byte("New"))
}
func (p *PostsController) Create(w ResponseWriter, r *Request) {
	w.Write([]byte("Create"))
}

func (p *PostsController) MemberPutActivateLater(w ResponseWriter, r *Request) {
	w.Write([]byte("ActivateLater " + r.GetParam("post_id")))
}

func (p *PostsController) Show(w ResponseWriter, r *Request) {
	w.Write([]byte("Show " + r.GetParam("post_id")))
}

func (p *PostsController) Update(w ResponseWriter, r *Request) {
	w.Write([]byte("Update " + r.GetParam("post_id")))
}

func (p *PostsController) Destroy(w ResponseWriter, r *Request) {
	w.Write([]byte("Destroy " + r.GetParam("post_id")))
}

func (p *PostsController) PostCreateSuper(w ResponseWriter, r *Request) {
	w.Write([]byte("CreateSuper"))
}

func testResourceExpectations(t *testing.T, r *ResourceDefinition, expectations []string) {
	t.Helper()
	routes := NewResource(r).Actions
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
		&ResourceDefinition{Controller: new(PostsController)},
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
		&ResourceDefinition{Controller: new(PostsController), PathNames: struct{ New, Edit string }{"nuevo", "editar"}},
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
		&ResourceDefinition{Controller: new(PostsController), Path: "/articles"},
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
		&ResourceDefinition{Controller: new(PostsController), Path: "/"},
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
		&ResourceDefinition{Controller: new(PostsController), Path: "/", Plural: "people", Singular: "person"},
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
		&ResourceDefinition{Controller: new(PostsController), ParamName: "article_id"},
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
		&ResourceDefinition{Controller: new(CommentsController)},
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

func TestResourceRoutes_SubResource(t *testing.T) {
	testResourceExpectations(
		t,
		&ResourceDefinition{
			Controller: new(PostsController),
			SubResources: []*ResourceDefinition{
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
