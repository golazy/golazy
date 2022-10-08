package lazyaction

import (
	"net/http"
	"net/url"
)

type Request struct {
	*http.Request
	Params url.Values
}

func (r *Request) GetParam(name string) string {
	return r.Params.Get(name)
}
