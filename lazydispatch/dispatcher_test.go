package lazydispatch

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

var langs = []string{"posts", "publicaciones", "veroffentlichung"}

type AuthorsController struct{}

func TestDispatcher(t *testing.T) {

	d := New()
	d.Draw(func(d *Scope) {

		for _, p := range langs {
			d.Resources(&PostsController{}).Path(p).Draw(func(posts *Scope) {
				posts.Resource(&AuthorsController{})
			})
		}

		d.Namespace("admin").Draw(func(admin *Scope) {

		})

		//d.Get("/patientes/:id").As("patient").To(&PostsController{}.Index)

	})

}

func ExampleDispatcher_Use() {
	d := New()
	d.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Println("Got request:", r.URL)
			next.ServeHTTP(w, r)
		})
	})
	s := httptest.NewServer(d)
	defer s.Close()
	r, _ := http.NewRequest("GET", s.URL+"/posts", nil)
	s.Client().Do(r)

	// Output:
	// Got request: /posts
}

type UsersController struct{}

func (UsersController) Index() string {
	return "index"
}

type PagesController struct{}

func (*PagesController) Index() string {
	return "index"
}
func (*PagesController) Show(page string) string {
	return "This is " + page
}

func ExampleDispatcher_Draw() {
	// 	type PagesController struct{}
	//
	//  func (*PagesController) Index() string {
	//  	return "index"
	//  }
	//
	//  func (*PagesController) Show(page string) string {
	//    return "This is " + page
	//  }

	dispatcher := New()
	dispatcher.Draw(func(routes *Scope) {
		routes.Resources(&PagesController{})
	})

	for _, r := range dispatcher.Routes {
		fmt.Printf("%s\t%s\t%s\n", r.URL, r.Target, r.Name)
	}

	// Output:
	// /pages	PagesController#Index	pages
	// /pages/:page_id	PagesController#Show	page
}
