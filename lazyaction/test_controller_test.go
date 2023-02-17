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

type ArticlesController struct {
}

func (p *ArticlesController) PostCreateSuper(w lazyaction.ResponseWriter, r *lazyaction.Request) {
	w.Write([]byte("CreateSuper"))
}

func (p *ArticlesController) Index(w lazyaction.ResponseWriter, r *lazyaction.Request) error {
	w.Write([]byte("Index"))
	return nil
}

func (p *ArticlesController) New(w lazyaction.ResponseWriter, r *lazyaction.Request) error {
	w.Write([]byte("New"))
	return nil

}
func (p *ArticlesController) Create(w lazyaction.ResponseWriter, r *lazyaction.Request) error {
	w.Write([]byte("Create"))
	return nil
}

func (p *ArticlesController) MemberPutActivateLater(w lazyaction.ResponseWriter, r *lazyaction.Request) error {
	w.Write([]byte("ActivateLater " + r.GetParam("post_id")))
	return nil
}

func (p *ArticlesController) Show(w lazyaction.ResponseWriter, r *lazyaction.Request) error {
	w.Write([]byte("Show " + r.GetParam("post_id")))
	return nil
}

func (p *ArticlesController) Update(w lazyaction.ResponseWriter, r *lazyaction.Request) error {
	w.Write([]byte("Update " + r.GetParam("post_id")))
	return nil
}

func (p *ArticlesController) Destroy(w lazyaction.ResponseWriter, r *lazyaction.Request) error {
	w.Write([]byte("Destroy " + r.GetParam("post_id")))
	return nil
}

type OpinionsController struct {
}

func (c *OpinionsController) Index(w lazyaction.ResponseWriter, r *lazyaction.Request) error {
	w.Write([]byte("Index"))
	return nil
}

func (c *OpinionsController) New(w lazyaction.ResponseWriter, r *lazyaction.Request) error {
	w.Write([]byte("New"))
	return nil

}

func (c *OpinionsController) Show(w lazyaction.ResponseWriter, r *lazyaction.Request) error {
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
