package apptest

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"runtime"
	"strings"
)

type Tester interface {
	Expect(...any) *Result
	Connect(ins ...any) *WSConn
}

type tester struct {
	t      T
	app    http.Handler
	server *httptest.Server
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

func New(t T, app http.Handler) Tester {
	t.Helper()
	return &tester{
		t:   t,
		app: app,
	}
}

func NewFull(t T, app http.Handler) Tester {
	t.Helper()
	tes := &tester{
		t:      t,
		app:    app,
		server: httptest.NewServer(app),
	}
	runtime.SetFinalizer(tes, func(my *tester) {
		my.server.Close()
	})
	return tes
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

type Response interface {
	Body() string
	Header() http.Header
	Code() int
}

type Result struct {
	t T
	//Response *httptest.ResponseRecorder
	Response Response
}

func (r *Result) Body(in string) *Result {
	r.t.Helper()
	if r.Response.Body() != in {
		r.t.Errorf("Expected body %q, got %q", in, r.Response.Body())
	}
	return r
}

func (r *Result) Contains(in string) *Result {
	r.t.Helper()
	if !strings.Contains(r.Response.Body(), in) {
		r.t.Errorf("Expected body to contain %q, got:\n%s\n", in, r.Response.Body())
	}
	return r
}

func (r *Result) Code(in int) *Result {
	r.t.Helper()
	if r.Response.Code() != in {
		r.t.Errorf("Expected code %d, got %d", in, r.Response.Code)
	}
	return r
}
func (r *Result) Header(key, in string) *Result {
	r.t.Helper()
	if r.Response.Header().Get(key) != in {
		r.t.Errorf("Expected header %q to be %q, got %q", key, in, r.Response.Header().Get(key))
	}
	return r
}

func (r *Result) Location(addr string) *Result {
	r.t.Helper()
	return r.Header("Location", addr)
}

func (r *Result) Headers() http.Header {
	return r.Response.Header()
}

type responseFromRecorder struct {
	*httptest.ResponseRecorder
}

func (r *responseFromRecorder) Body() string {
	return r.ResponseRecorder.Body.String()
}

func (r *responseFromRecorder) Header() http.Header {
	return r.ResponseRecorder.Header()
}

func (r *responseFromRecorder) Code() int {
	return r.ResponseRecorder.Code
}

type responseFromResponse struct {
	*http.Response
}

func (r *responseFromResponse) Body() string {
	buf := &bytes.Buffer{}
	io.Copy(buf, r.Response.Body)
	r.Response.Body.Close()
	return buf.String()
}

func (r *responseFromResponse) Header() http.Header {
	return r.Response.Header
}

func (r *responseFromResponse) Code() int {
	return r.Response.StatusCode
}

func (t *tester) Expect(ins ...any) *Result {
	t.t.Helper()
	r := &Request{}
	fillRequest(r, ins...)

	if app, ok := t.app.(app); ok {
		app.Init()
	}

	if t.server == nil {

		req := httptest.NewRequest(r.method, r.URL, r)
		req.Header = r.Headers
		rec := httptest.NewRecorder()

		t.app.ServeHTTP(rec, req)

		result := &Result{
			t:        t.t,
			Response: &responseFromRecorder{rec},
		}

		return result
	}

	req, err := http.NewRequest(r.method, t.server.URL+r.URL, r)
	if err != nil {
		t.t.Fatal(err)
	}

	res, err := t.server.Client().Do(req)
	if err != nil {
		t.t.Fatal(err)
	}
	return &Result{
		t:        t.t,
		Response: &responseFromResponse{res},
	}

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
