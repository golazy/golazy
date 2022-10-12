package lazyaction

import "net/http"

type MultiArgsController struct{}

type User struct {
	Name string
}

func (mac *MultiArgsController) Show(string, ResponseWriter, *http.Request) interface{} {

	return "hello"
}

type UsersController struct {
}

func (uc *UsersController) Show(w ResponseWriter, r *Request) {
	w.Write([]byte("Showing user " + r.GetParam("user_id")))
}

type DevicesController struct {
}

func (dc *DevicesController) Show(w ResponseWriter, r *Request) {
	w.Write([]byte("Showing device " + r.GetParam("device_id")))
}

type PostsController struct {
}

func (p *PostsController) PostCreateSuper(w ResponseWriter, r *Request) {
	w.Write([]byte("CreateSuper"))
}

func (p *PostsController) Index(w ResponseWriter, r *Request) error {
	w.Write([]byte("Index"))
	return nil
}

func (p *PostsController) New(w ResponseWriter, r *Request) error {
	w.Write([]byte("New"))
	return nil

}
func (p *PostsController) Create(w ResponseWriter, r *Request) error {
	w.Write([]byte("Create"))
	return nil
}

func (p *PostsController) MemberPutActivateLater(w ResponseWriter, r *Request) error {
	w.Write([]byte("ActivateLater " + r.GetParam("post_id")))
	return nil
}

func (p *PostsController) Show(w ResponseWriter, r *Request) error {
	w.Write([]byte("Show " + r.GetParam("post_id")))
	return nil
}

func (p *PostsController) Update(w ResponseWriter, r *Request) error {
	w.Write([]byte("Update " + r.GetParam("post_id")))
	return nil
}

func (p *PostsController) Delete(w ResponseWriter, r *Request) error {
	w.Write([]byte("Delete " + r.GetParam("post_id")))
	return nil
}
