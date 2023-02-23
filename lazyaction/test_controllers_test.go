package lazyaction

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func ActionHandler(id string) string {
	return id
}

type StringHandler string

func (h StringHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(h))
}

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

func (p *PostsController) Index(w http.ResponseWriter, r *http.Request) {
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

func (p *PostsController) GetAbout() string {
	return "about"
}

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

func (p *ArticlesController) PostCreateSuper(w ResponseWriter, r *Request) {
	w.Write([]byte("CreateSuper"))
}

func (p *ArticlesController) Index(w ResponseWriter, r *Request) error {
	w.Write([]byte("Index"))
	return nil
}

func (p *ArticlesController) New(w ResponseWriter, r *Request) error {
	w.Write([]byte("New"))
	return nil

}
func (p *ArticlesController) Create(w ResponseWriter, r *Request) error {
	w.Write([]byte("Create"))
	return nil
}

func (p *ArticlesController) MemberPutActivateLater(w ResponseWriter, r *Request) error {
	w.Write([]byte("ActivateLater " + r.GetParam("post_id")))
	return nil
}

func (p *ArticlesController) Show(w ResponseWriter, r *Request) error {
	w.Write([]byte("Show " + r.GetParam("post_id")))
	return nil
}

func (p *ArticlesController) Update(w ResponseWriter, r *Request) error {
	w.Write([]byte("Update " + r.GetParam("post_id")))
	return nil
}

func (p *ArticlesController) Destroy(w ResponseWriter, r *Request) error {
	w.Write([]byte("Destroy " + r.GetParam("post_id")))
	return nil
}

type OpinionsController struct {
}

func (c *OpinionsController) Index(w ResponseWriter, r *Request) error {
	w.Write([]byte("Index"))
	return nil
}

func (c *OpinionsController) New(w ResponseWriter, r *Request) error {
	w.Write([]byte("New"))
	return nil

}

func (c *OpinionsController) Show(w ResponseWriter, r *Request) error {
	w.Write([]byte("Show " + r.GetParam("comment_id")))
	return nil
}

type ReviewsController struct {
}

func (c *ReviewsController) Index(w ResponseWriter, r *Request) error {
	w.Write([]byte("Index"))
	return nil
}

func (c *ReviewsController) Show(w ResponseWriter, r *Request) error {
	w.Write([]byte("Show " + r.GetParam("review_id")))
	return nil
}

type Page struct {
	Title, Body string
}

type PagesController struct {
}

func (c *PagesController) Show(w ResponseWriter, r *Request) error {
	w.Write([]byte("Show " + r.GetParam("page_id")))
	return nil
}

type EmptyController struct{}

// TestController ActionHandlerController
type TestController struct{}

func (TestController) GetInHttpResponseWriter(w http.ResponseWriter) {
	w.Write([]byte("InHttpResponseWriter"))
}

func (TestController) GetInResponseWriter(w http.ResponseWriter) {
	w.Write([]byte("InResponseWriter"))
}

func (TestController) GetOutString() string {
	return "OutString"
}

func (TestController) MemberGetInString(s string) string {
	return s
}
func (TestController) MemberGetInStringString(s1, s2 string) string {
	return strings.Join([]string{s1, s2}, ",")
}

func (TestController) GetOutBytes() []byte {
	return []byte("OutBytes")
}

func (TestController) GetOutError() error {
	return fmt.Errorf("OutError")
}

func (TestController) GetOutInt() (string, int) {
	return "OutInt", 204
}

func (TestController) Show(id string) string {
	return id
}

func (TestController) GetRedirect(ctx *context.Context) {
	//ctx.Redirect("http://google.com", 301)
}

func (TestController) MemberGetSetSession(id string, s *Session) {
	//s.Set("id", id)
}

func (TestController) GetSession(s *Session) string {
	//id := s.Get("id")
	//fmt.Println(id)
	return "asdf"
}

func (TestController) SetError(s *Session) {
	//s.SetError(fmt.Errorf("error"))
}

func (TestController) GetGetError(s *Session) string {
	return "asdf"
}

func (TestController) Delete() string {
	return "Delete"
}

type TestUser string

type UserProvider struct {
}

func (u *UserProvider) GenUser(r *http.Request) *TestUser {
	user := TestUser("user")
	return &user
}

type GeneratorController struct {
	UserProvider
}

func (g *GeneratorController) Index(u *TestUser) string {
	return string(*u)
}

type LayoutController struct {
}

func (g *LayoutController) RenderLayout(content []byte) io.WriterTo {
	return bytes.NewBufferString("--" + string(content) + "--")
}

func (g *LayoutController) Index() string {
	return "index"
}

type BasicLayout struct {
}

func (g *BasicLayout) RenderLayout(content []byte) string {
	return "--" + string(content) + "--"
}

type PagesWithLayout struct {
	BasicLayout
}

func (g *PagesWithLayout) Index() string {
	return "embebed index"
}
