package lazyaction

import (
	"context"
	"net/http"
)

type Context struct {
	context.Context
	r *http.Request
	w http.ResponseWriter
	Session
	status  int
	headers http.Header
}

func (c *Context) Redirect(url string, status int) {
	if c.headers == nil {
		c.headers = http.Header{}
	}
	c.headers.Set("Location", url)
	c.status = status
}
