package lazydispatch

import (
	"net/http/httptest"
	"testing"

	"golazy.dev/lazydispatch/test_controllers"
)

func TestRouter_Simple(t *testing.T) {

	dispatcher := New()

	dispatcher.Draw(func(r *Scope) {
		r.Resources(&test_controllers.PostsController{})
	})

	for _, r := range dispatcher.Routes {
		t.Log(r.String())
	}
	w := httptest.NewRecorder()
	path := dispatcher.PathFor("reject_post", 4)
	if path != "/posts/4/reject" {
		t.Fatalf("expected %s, got %s", "/posts/4/reject", path)
	}
	r := httptest.NewRequest("PUT", dispatcher.PathFor("reject_post", 4), nil)
	dispatcher.ServeHTTP(w, r)
	if w.Body.String() != "member_put_reject" {
		t.Errorf("expected %s, got %s", "member_put_reject", w.Body.String())
	}

}

func TestRouter_Resources(t *testing.T) {
	// router := NewRouter()
	//
	//	router.Draw(func(r *RouteDrawer){
	//		r.Draw(Resources{
	//			Controller: &PostController{},
	//		}, func(r *RouteDrawer) {
	//			r.Draw(Resources{
	//				Controller: &PostController{},
	//			})
	//		}))
	//
	// })
	//
	//	router.Add(Resources{
	//		Controller: &PostController{},
	//	}, func(r *Router))
}
