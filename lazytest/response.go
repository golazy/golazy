package lazytest

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
)

// Response is a test response
type Response struct {
	at *AppTest
	r  *http.Request
	w  *httptest.ResponseRecorder
}

// Request makes a request to the app
func (at *AppTest) Request(method string, body io.Reader, path ...any) *Response {
	var p string
	at.t.Helper()
	at.boot()

	if len(path) == 0 {
		at.t.Fatalf("Request path requires at least one argument")
	}
	p, isString := path[0].(string)
	if len(path) != 1 || !isString || !strings.HasPrefix(p, "/") {
		p = at.PathFor(path...)
	} else {
		p = path[0].(string)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(method, p, nil).WithContext(at.appCtx)

	at.Handler().ServeHTTP(w, r)

	response := &Response{
		at: at,
		r:  r,
		w:  w,
	}
	return response
}

// ExpectCode expects a specific status code
func (r *Response) ExpectCode(code int) *Response {
	r.at.t.Helper()
	if r.w.Code != code {
		r.at.t.Errorf("expected code %d got %d", code, r.w.Code)
	}
	return r
}

// Contains expects the response to contain a specific string
func (r *Response) Contains(s string) *Response {
	r.at.t.Helper()
	if !strings.Contains(r.Body(), s) {
		r.at.t.Errorf("expected response to contain %q. Got: %q", s, r.Body())
	}
	return r
}

// Body returns the body of the response
func (r *Response) Body() string {
	return r.w.Body.String()
}

// Header returns the header of the response
func (r *Response) Header() http.Header {
	return r.w.Header()
}
