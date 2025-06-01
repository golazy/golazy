package response_editor

import (
	"compress/gzip"
	"io"
	"net/http"
)

type BufResponse struct {
}

func asdf(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w = pipe(w, compress)
		w = pipe(w, md5)

		h.ServeHTTP(w, r)
	})
}

func pipe(w http.ResponseWriter, fn func()) {

}

func compress(h http.Header, in io.Reader, out io.Writer) error {
	h.Set("Content-Encoding", "gzip")
	writer := gzip.NewWriter(out)
	n, err := io.Copy(writer, in)
	if err != nil {

	}
	err = writer.Close()
	if err != nil {

	}

}
