package lazyaction

import (
	"net/http"
)

type Request struct {
	*http.Request
}

func (r *Request) GetParam(name string) string {
	return ""
}
