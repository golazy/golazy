package actionrecorder

import (
	"bytes"
	"net/http"
)

type Recorder struct {
	http.ResponseWriter
	B *bytes.Buffer
	S int
}

func New(w http.ResponseWriter) *Recorder {
	return &Recorder{
		ResponseWriter: w,
		B:              &bytes.Buffer{},
	}
}

func (r *Recorder) Write(b []byte) (int, error) {
	return r.B.Write(b)
}
func (r *Recorder) WriteHeader(statusCode int) {
	r.S = statusCode
}

func (r *Recorder) Bytes() []byte {
	d := make([]byte, r.B.Len())
	copy(d, r.B.Bytes())
	return d
}

func (r *Recorder) Send() {
	if r.S != 0 {
		r.ResponseWriter.WriteHeader(r.S)
	}
	r.B.WriteTo(r.ResponseWriter)
}
