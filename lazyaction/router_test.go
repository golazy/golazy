package lazyaction_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golazy.dev/lazyaction"
)

type StringHandler string

func (h StringHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(h))
}

func TestRouterRoute(t *testing.T) {

	router := lazyaction.Routes{}

	expect := func(path, expected string) {
		req, err := http.NewRequest("GET", path, nil)
		if err != nil {
			t.Fatal(err)
		}
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Body.String() != expected {
			t.Errorf("expected %q to return %q got %q", path, expected, rr.Body.String())
		}
	}

	router.Route("/", StringHandler("root"))
	router.Route("/hi", StringHandler("hi"))
	expect("/", "root")
	expect("/", "hi")

	t.Error(router.String())

}

func TestRouterResource(t *testing.T) {

	router := lazyaction.Routes{}

	expect := func(path, expected string) {
		req, err := http.NewRequest("GET", path, nil)
		if err != nil {
			t.Fatal(err)
		}
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Body.String() != expected {
			t.Errorf("expected %q to return %q got %q", path, expected, rr.Body.String())
		}
	}

	router.Resource(&TestController{})
	expect("/", "root")
	expect("/", "hi")

	t.Error(router.String())

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

type Session struct{}

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
