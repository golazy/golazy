package router

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
