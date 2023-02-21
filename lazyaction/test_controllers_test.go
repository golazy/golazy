package lazyaction

import (
	"time"
)

type InternalController struct {
}

func (i *InternalController) Index(w ResponseWriter, r *Request) {
	w.Write([]byte("Index"))
}

type Comment struct {
	Comment   string
	Author    string
	CreatedAt time.Time
}

type CommentsController struct {
	RestController[Comment, MemStore[Comment]]
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
