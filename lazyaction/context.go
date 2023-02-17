package lazyaction

import (
	"context"
	"net/http"

	"github.com/timewasted/go-accept-headers"
)

type Context struct {
	context.Context
	Request         *http.Request
	ResponseWritter http.ResponseWriter
	Session
	status  int
	headers http.Header
	replied bool
	// Router
	// Assets
	// Views
}

func (c *Context) GetHeader(h string) string {
	return c.Request.Header.Get(h)
}

func (c *Context) alreadyReplied() {
	panic("already replied")
}

func (c *Context) Redirect(url string, status int) {
	if c.replied {
		c.alreadyReplied()
		return
	}

	if c.headers == nil {
		c.headers = http.Header{}
	}
	c.headers.Set("Location", url)
	c.status = status
	c.replied = true
}

func (c *Context) Render(data ...any) {

	mime, _ := accept.Negotiate(c.GetHeader("Accept"), "text/html", "application/json", "text/plain")

	panic(mime)
}

func (c *Context) WriteString(data string) {
	c.Write([]byte(data))
}

func (c *Context) Write(data []byte) {
	c.ResponseWritter.Write(data)
}
