package lazyaction

import (
	"context"
	"io"
	"net/http"

	"github.com/timewasted/go-accept-headers"
)

func newContext(w http.ResponseWriter, r *http.Request) *Context {
	c := &Context{
		Context:        r.Context(),
		Request:        r,
		ResponseWriter: w,
	}
	return c
}

type Context struct {
	context.Context
	Request        *http.Request
	ResponseWriter http.ResponseWriter
	Session
	status  int
	headers http.Header
	replied bool
	// Router
	// Assets
	// Views
}

func (c *Context) PathTo(args ...any) string {
	panic("not implemented")
}

func (c *Context) SendFile(filename string, data io.Reader) {
	c.ResponseWriter.Header().Set("Content-Disposition", "attachment; filename=\""+filename)
	io.Copy(c.ResponseWriter, data)
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
	c.ResponseWriter.Write(data)
}
