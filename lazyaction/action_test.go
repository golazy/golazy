package lazyaction

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golazy.dev/lazysupport"
)

type EmptyController struct{}

// ActionController ActionHandlerController
type ActionController struct{}

func (ActionController) GetInHttpResponseWriter(w http.ResponseWriter) {
	w.Write([]byte("InHttpResponseWriter"))
}

func (ActionController) GetInResponseWriter(w ResponseWriter) {
	w.WriteString("InResponseWriter")
}

func (ActionController) GetOutString() string {
	return "OutString"
}

func (ActionController) MemberGetInString(s string) string {
	return s
}
func (ActionController) MemberGetInStringString(s1, s2 string) string {
	return strings.Join([]string{s1, s2}, ",")
}

func (ActionController) GetOutBytes() []byte {
	return []byte("OutBytes")
}

func (ActionController) GetOutError() error {
	return fmt.Errorf("OutError")
}

func (ActionController) GetOutInt() (string, int) {
	return "OutInt", 204
}

func (ActionController) Show(id string) string {
	return id
}

func (ActionController) GetRedirect(ctx *Context) {
	ctx.Redirect("http://google.com", 301)
}

func (ActionController) MemberGetSetSession(id string, s *Session) {
	s.Set("id", id)
}

func (ActionController) GetGetSession(s *Session) string {
	id := s.Get("id")
	fmt.Println(id)
	return id.(string)
}

func (ActionController) GetSetError(s *Session) {
	s.SetError(fmt.Errorf("error"))
}

func (ActionController) GetGetError(s *Session) string {
	err := s.GetError()
	return err
}

func TestActionSession(t *testing.T) {
	router := NewRouter()
	router.AddResourceDefinition(&ResourceDefinition{
		Controller: new(ActionController),
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/action/33/set_session", nil)

	router.ServeHTTP(w, r)

	r = httptest.NewRequest("GET", "/action/get_session", nil)
	r.Header.Set("Cookie", w.Header().Get("Set-Cookie"))
	w = httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Body.String() != "33" {
		t.Fatal(w.Body.String())
	}
}

func TestActionSessionFlashError(t *testing.T) {
	router := NewRouter()
	router.AddResourceDefinition(&ResourceDefinition{
		Controller: new(ActionController),
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/action/set_error", nil)

	router.ServeHTTP(w, r)

	r = httptest.NewRequest("GET", "/action/get_error", nil)
	r.Header.Set("Cookie", w.Header().Get("Set-Cookie"))
	w = httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Body.String() != "error" {
		t.Fatal(w.Body.String())
	}
}

func TestActionParams(t *testing.T) {

	r := NewResource(&ResourceDefinition{
		Controller: new(EmptyController),
		SubResources: []*ResourceDefinition{
			{Controller: new(ActionController)},
		},
	})

	router := NewRouter()
	router.AddResource(r)

	t.Log(r.Routes())

	test := func(method, path, expectation string, status int) {

		t.Helper()
		if expectation == "" {
			expectation = method
		}
		if status == 0 {
			status = 200
		}

		if path == "" {
			path = lazysupport.Underscorize(method)
		}

		path = "/empty/42/action/" + path

		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", path, nil)

		router.ServeHTTP(w, r)
		if status != 0 && w.Code != status {
			t.Errorf("Expected %s to return status %d, got %d in %s", method, status, w.Code, path)
		}
		if strings.TrimSpace(w.Body.String()) != strings.TrimSpace(expectation) {
			t.Errorf("Expected %s to return %q, got %q in %s", method, expectation, w.Body.String(), path)
		}
	}

	//test("InHttpResponseWriter", "", "", 0)
	//test("InResponseWriter", "", "", 0)
	//test("OutString", "", "", 0)
	//test("OutBytes", "", "", 0)
	//test("OutError", "", "", 500)
	//test("OutInt", "", "", 204)
	//test("Show", "69", "69", 200)
	//test("InString", "55/in_string", "55", 0)
	//test("InStringString", "44/in_string_string", "44,42", 0)
	test("Redirect", "", " ", 301)

}
func TestExtract(t *testing.T) {

	expect := func(path string, stringArg int, paramsPosition []int, expectation string) {
		result := UrlExtractor(path).Extract(stringArg, paramsPosition)
		if result != expectation {
			t.Errorf("Expected %q with (%d, %v) to return %q, got %q", path, stringArg, paramsPosition, expectation, result)
		}

	}

	expect("/post/33/comments/44", 0, []int{1, 3}, "44")
	expect("/post/33/comments/44", 1, []int{1, 3}, "33")
	expect("/post/33/comments/44", 2, []int{1, 3}, "")
	expect("/123", 0, []int{0}, "123")
	expect("/index", 0, []int{0}, "index")

	//	/post/33/comments/44     CommentsController#Show(id string)                 1              [1,3]            3, then 1

}
