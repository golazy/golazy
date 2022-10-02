package lazyaction

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

func TestBuildResourceRoutes(t *testing.T) {
	rr := buildResourceRoutes(&PostsController{})
	testMatch := func(exp string) {
		ok, err := regexp.MatchString(exp, rr.String())
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Errorf("route %q not defined", exp)
			t.Log(rr.String())
		}
	}

	testMatch("POST /posts .* PostsController#Create")
	testMatch("GET /posts .* PostsController#Index")
	testMatch("GET /posts/new .* PostsController#New")
	testMatch("POST /posts/create_super .* PostsController#PostCreateSuper")
	testMatch("DELETE /posts/:id .* PostsController#Delete")
	testMatch("GET /posts/:id .* PostsController#Show")
	testMatch("PUT /posts/:id .* PostsController#Update")
	testMatch("ATCH /posts/:id .* PostsController#Update")
	testMatch("PUT /posts/:id/activate_later .* PostsController#PutActivateLater")
}

func TestServeHTTP(t *testing.T) {
	routes := buildResourceRoutes(&PostsController{})

	testRequest := func(verb, path, expectation string) {
		req, err := http.NewRequest(verb, path, nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()

		routes.ServeHTTP(rr, req)

		body := strings.TrimSpace(rr.Body.String())
		if body != expectation {
			t.Errorf("Expected %s %s to be %s. Got %q", verb, path, expectation, body)
		}
	}

	testRequest("GET", "/posts/39", "Show 39")
	testRequest("PUT", "/posts/39/activate_later", "ActivateLater 39")
	testRequest("GET", "/posts", "Index")
	testRequest("GET", "/", "404 page not found")
	testRequest("POST", "/posts", "Create")
	testRequest("POST", "/posts/create_super", "CreateSuper")
	testRequest("GET", "/posts/new", "New")
	testRequest("PUT", "/posts/39", "Update 39")
	testRequest("PATCH", "/posts/39", "Update 39")
	testRequest("DELETE", "/posts/39", "Delete 39")
	testRequest("GET", "/posts/111/", "<a href=\"/111\">Found</a>.") // Redirects trailing slashes

}

type PostsController struct {
}

func (p *PostsController) PostCreateSuper(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("CreateSuper"))

}

func (p *PostsController) Index(w http.ResponseWriter, r *http.Request) error {
	w.Write([]byte("Index"))
	return nil
}

func (p *PostsController) New(w http.ResponseWriter, r *http.Request) error {
	w.Write([]byte("New"))
	return nil

}
func (p *PostsController) Create(w http.ResponseWriter, r *http.Request) error {
	w.Write([]byte("Create"))
	return nil
}

func (p *PostsController) PutActivateLater(id string, w http.ResponseWriter, r *http.Request) error {
	w.Write([]byte("ActivateLater " + id))
	return nil
}

func (p *PostsController) Show(id string, w http.ResponseWriter, r *http.Request) error {
	w.Write([]byte("Show " + id))
	return nil
}

func (p *PostsController) Update(id string, w http.ResponseWriter, r *http.Request) error {
	w.Write([]byte("Update " + id))
	return nil
}

func (p *PostsController) Delete(id string, w http.ResponseWriter, r *http.Request) error {
	w.Write([]byte("Delete " + id))
	return nil
}

func main() {
	http.ListenAndServe(":4000", nil)
}
