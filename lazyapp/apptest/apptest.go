package apptest

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
)

type Tester struct {
	t   T
	app http.Handler
}

type app interface {
	Init()
}

type T interface {
	Helper()
	Errorf(format string, args ...interface{})
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
}

func New(t T, app http.Handler) *Tester {
	t.Helper()
	return &Tester{
		t:   t,
		app: app,
	}
}

type Request struct {
	URL       string
	Body      io.Reader
	Headers   http.Header
	method    string
	readIndex int64
}

func (r *Request) Read(b []byte) (n int, err error) {
	if r.Body == nil {
		return 0, io.EOF
	}
	return r.Body.Read(b)
}

type Expectation struct {
	Body     string
	Contains string
	Code     int
	Headers  http.Header
}

type Result struct {
	t   T
	rec *httptest.ResponseRecorder
}

func (r *Result) Body(in string) *Result {
	r.t.Helper()
	if r.rec.Body.String() != in {
		r.t.Errorf("Expected body %q, got %q", in, r.rec.Body.String())
	}
	return r
}

func (r *Result) Contains(in string) *Result {
	r.t.Helper()
	if !strings.Contains(r.rec.Body.String(), in) {
		r.t.Errorf("Expected body to contain %q, got:\n%s\n", in, r.rec.Body.String())
	}
	return r
}

func (r *Result) Code(in int) *Result {
	r.t.Helper()
	if r.rec.Code != in {
		r.t.Errorf("Expected code %d, got %d", in, r.rec.Code)
	}
	return r
}
func (r *Result) Header(key, in string) *Result {
	r.t.Helper()
	if r.rec.Header().Get(key) != in {
		r.t.Errorf("Expected header %q to be %q, got %q", key, in, r.rec.Header().Get(key))
	}
	return r
}

func (t *Tester) Expect(ins ...any) *Result {
	r := &Request{}
	fillRequest(r, ins...)

	req := httptest.NewRequest(r.method, r.URL, r)
	rec := httptest.NewRecorder()

	if app, ok := t.app.(app); ok {
		app.Init()
	}
	t.app.ServeHTTP(rec, req)

	result := &Result{
		t:   t.t,
		rec: rec,
	}

	return result
}

func fillRequest(req *Request, in ...any) {
	for _, v := range in {
		switch in := v.(type) {
		case string:
			parts := strings.SplitN(in, " ", 2)
			if len(parts) == 2 {
				req.method = parts[0]
				req.URL = parts[1]
			} else {
				req.URL = in
			}

		case *url.URL:
			req.URL = in.String()
		case []byte:
			req.Body = bytes.NewBuffer(in)
		case io.Reader:
			req.Body = in
		case http.Header:
			req.Headers = in
		default:
			panic(fmt.Sprintf("%T is not a valid request type", in))
		}
	}
}
