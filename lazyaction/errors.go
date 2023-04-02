package lazyaction

import (
	"bytes"
	"errors"
	"io"
	"net/http"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrNotAuthorized = errors.New("not authorized")
)

type Result http.Handler

func Redirect(url string, status ...int) *redirect {
	r := &redirect{url, 307}
	if len(status) > 0 {
		r.Status = status[0]
	}
	return r
}

type redirect struct {
	URL    string
	Status int
}

func (r *redirect) Error() string {
	return "redirect"
}
func (r *redirect) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if r.Status == 0 {
		r.Status = http.StatusFound
	}
	http.Redirect(w, req, r.URL, r.Status)
}

type htmlError string

func (h htmlError) Error() string {
	return string(h)
}

func (h htmlError) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	io.WriteString(w, string(h))
}

func HTMLError(data ...io.WriterTo) htmlError {
	buf := &bytes.Buffer{}
	for _, d := range data {
		d.WriteTo(buf)
	}
	return htmlError(buf.String())
}
