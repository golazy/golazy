package lazyaction

import (
	"bytes"
	"net/http"
	"strconv"
)

type actionRecorder struct {
	http.ResponseWriter
	H http.Header
	B *bytes.Buffer
	S int
}

func newActionRecorder(w http.ResponseWriter) *actionRecorder {
	return &actionRecorder{
		ResponseWriter: w,
		H:              http.Header{},
		B:              &bytes.Buffer{},
	}
}

func (r *actionRecorder) Write(b []byte) (int, error) {
	return r.B.Write(b)
}
func (r *actionRecorder) WriteHeader(statusCode int) {
	r.S = statusCode
}

func (r *actionRecorder) Bytes() []byte {
	d := make([]byte, r.B.Len())
	copy(d, r.B.Bytes())
	return d
}

func (r *actionRecorder) Update(w http.ResponseWriter) {
	// Copy headers
	for k, v := range r.H {
		w.Header()[k] = v
	}
	// Copy status
	if r.S != 0 {
		w.WriteHeader(r.S)
	}

}

func (r *actionRecorder) Send() {
	r.ResponseWriter.Header().Set("Content-Length", strconv.Itoa(r.B.Len()))
	if r.S != 0 {
		r.ResponseWriter.WriteHeader(r.S)
	}
	r.B.WriteTo(r.ResponseWriter)
}
