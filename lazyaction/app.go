package lazyaction

import (
	"net/http"
)

type PostsController struct {
}

func (p *PostsController) Index(w http.ResponseWriter, r *http.Request) error {
	w.Write([]byte("This is index"))
	return nil
}

func (p *PostsController) New(w http.ResponseWriter, r *http.Request) error {
	w.Write([]byte("This is new"))
	return nil

}
func (p *PostsController) Create(w http.ResponseWriter, r *http.Request) error {
	w.Write([]byte("This is index"))
	return nil
}

func (p *PostsController) Show(id string, w http.ResponseWriter, r *http.Request) error {
	w.Write([]byte("Showing " + id))
	return nil
}

func (p *PostsController) Update(id string, w http.ResponseWriter, r *http.Request) error {
	w.Write([]byte("Updating " + id))
	return nil
}

func (p *PostsController) Delete(id string, w http.ResponseWriter, r *http.Request) error {
	w.Write([]byte("Deleting " + id))
	return nil
}

func main() {
	Route("/", new(PostsController))
	http.ListenAndServe(":4000", nil)
}
