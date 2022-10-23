package lazyaction_test

import (
	"net/http"
	"time"

	"golazy.dev/lazyaction"
)

type MultiArgsController struct{}

type User struct {
	Name string
}

func (mac *MultiArgsController) Show(string, lazyaction.ResponseWriter, *http.Request) interface{} {

	return "hello"
}

type UsersController struct {
}

func (uc *UsersController) Show(w lazyaction.ResponseWriter, r *lazyaction.Request) {
	w.Write([]byte("Showing user " + r.GetParam("user_id")))
}

type DevicesController struct {
}

func (dc *DevicesController) Show(w lazyaction.ResponseWriter, r *lazyaction.Request) {
	w.Write([]byte("Showing device " + r.GetParam("device_id")))
}

type Post struct {
	Id        string
	Slug      string
	Title     string
	Body      string
	CreatedAt time.Time
	Author    *User
}

type PostsController struct {
}

func (p *PostsController) PostCreateSuper(w lazyaction.ResponseWriter, r *lazyaction.Request) {
	w.Write([]byte("CreateSuper"))
}

func (p *PostsController) Index(w lazyaction.ResponseWriter, r *lazyaction.Request) error {
	w.Write([]byte("Index"))
	return nil
}

func (p *PostsController) New(w lazyaction.ResponseWriter, r *lazyaction.Request) error {
	w.Write([]byte("New"))
	return nil

}
func (p *PostsController) Create(w lazyaction.ResponseWriter, r *lazyaction.Request) error {
	w.Write([]byte("Create"))
	return nil
}

func (p *PostsController) MemberPutActivateLater(w lazyaction.ResponseWriter, r *lazyaction.Request) error {
	w.Write([]byte("ActivateLater " + r.GetParam("post_id")))
	return nil
}

func (p *PostsController) Show(w lazyaction.ResponseWriter, r *lazyaction.Request) error {
	w.Write([]byte("Show " + r.GetParam("post_id")))
	return nil
}

func (p *PostsController) Update(w lazyaction.ResponseWriter, r *lazyaction.Request) error {
	w.Write([]byte("Update " + r.GetParam("post_id")))
	return nil
}

func (p *PostsController) Destroy(w lazyaction.ResponseWriter, r *lazyaction.Request) error {
	w.Write([]byte("Destroy " + r.GetParam("post_id")))
	return nil
}

type CommentsController struct {
}

func (c *CommentsController) Index(w lazyaction.ResponseWriter, r *lazyaction.Request) error {
	w.Write([]byte("Index"))
	return nil
}

func (c *CommentsController) New(w lazyaction.ResponseWriter, r *lazyaction.Request) error {
	w.Write([]byte("New"))
	return nil

}

func (c *CommentsController) Show(w lazyaction.ResponseWriter, r *lazyaction.Request) error {
	w.Write([]byte("Show " + r.GetParam("comment_id")))
	return nil
}

type ReviewsController struct {
}

func (c *ReviewsController) Index(w lazyaction.ResponseWriter, r *lazyaction.Request) error {
	w.Write([]byte("Index"))
	return nil
}

func (c *ReviewsController) Show(w lazyaction.ResponseWriter, r *lazyaction.Request) error {
	w.Write([]byte("Show " + r.GetParam("review_id")))
	return nil
}

type Page struct {
	Title, Body string
}

type PagesController struct {
}

func (c *PagesController) Show(w lazyaction.ResponseWriter, r *lazyaction.Request) error {
	w.Write([]byte("Show " + r.GetParam("page_id")))
	return nil
}
