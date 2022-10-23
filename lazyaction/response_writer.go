package lazyaction

import "net/http"

type ResponseWriter struct {
	http.ResponseWriter
}

func (w ResponseWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}
