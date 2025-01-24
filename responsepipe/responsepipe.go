package responsepipe

import (
	"io"
	"net/http"
)

type response struct {
	status <-chan(int)
	header http.Header
	io.Writer
	original http.ResponseWriter
}

func (r *response) WriteHeader(status int) {
	r.status = status
}
func (r *response) Header() http.Header {
	return r.header
}

func New(w http.ResponseWriter, r *http.Request, h http.Handler) (io.Reader, http.ResponseWriter, int, error) {

	reader, writer := io.Pipe()

	response := &response{
		status: make(chan int, 1),
		Writer: writer,
		header: http.Header{},
	}

	go func(){
		h.ServeHTTP(response, r)
		writer.Close()
	}

	status, ok := <-response.status
	if !ok {
		return 
	}




	return reader, w, response.status, nil
	writer.Close()
}
