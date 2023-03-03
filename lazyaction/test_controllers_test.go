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

func (i *InternalController) Index(w http.ResponseWriter, r *http.Request) {
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

func (p *PostsController) New(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("New"))
}
func (p *PostsController) Edit(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("New"))
}
func (p *PostsController) Create(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Create"))
}

func (p *PostsController) MemberPutActivateLater(id string) string {
	return "ActivateLater " + id
}

func (p *PostsController) Show(id string) string {
	return "Show " + id
}

func (p *PostsController) Update(id string) string {
	return "Update " + id
}

func (p *PostsController) Destroy(id string) string {
	return "Destroy " + id
}

func (p *PostsController) PostCreateSuper(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("CreateSuper"))
}

func (p *PostsController) GetAbout() string {
	return "about"
}

type MultiArgsController struct{}

type User struct {
	Name string
}

func (mac *MultiArgsController) Show(string, http.ResponseWriter, *http.Request) any {

	return "hello"
}

type UsersController struct {
}

func (uc *UsersController) Show(id string) string {
	return "Showing user " + id
}

type DevicesController struct {
}

func (dc *DevicesController) Show(id string) string {
	return "Showing device " + id
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

func (p *ArticlesController) PostCreateSuper(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("CreateSuper"))
}

func (p *ArticlesController) Index(w http.ResponseWriter, r *http.Request) error {
	w.Write([]byte("Index"))
	return nil
}

func (p *ArticlesController) New(w http.ResponseWriter, r *http.Request) error {
	w.Write([]byte("New"))
	return nil

}
func (p *ArticlesController) Create(w http.ResponseWriter, r *http.Request) error {
	w.Write([]byte("Create"))
	return nil
}

func (p *ArticlesController) MemberPutActivateLater(id string) (string, error) {
	return "ActivateLater " + id, nil
}

func (p *ArticlesController) Show(id string) (string, error) {
	return "Show " + id, nil
}

func (p *ArticlesController) Update(id string) (string, error) {
	return "Update " + id, nil
}

func (p *ArticlesController) Destroy(id string) (string, error) {
	return "Destroy " + id, nil
}

type OpinionsController struct {
}

func (c *OpinionsController) Index(id string) (string, error) {
	return "Index", nil
}

func (c *OpinionsController) New(id string) (string, error) {
	return "New", nil

}

func (c *OpinionsController) Show(id string) (string, error) {
	return "Show " + id, nil
}

type ReviewsController struct {
}

func (c *ReviewsController) Index(w http.ResponseWriter, r *http.Request) error {
	w.Write([]byte("Index"))
	return nil
}

func (c *ReviewsController) Show(id string) (string, error) {
	return "Show " + id, nil
}

type Page struct {
	Title, Body string
}

type PagesController struct {
}

func (c *PagesController) Show(id string) (string, error) {
	return "Show " + id, nil
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
