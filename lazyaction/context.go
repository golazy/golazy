package lazyaction

import (
	"context"
	"encoding/json"
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
	replied        bool
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

func (c *Context) JSON(data any) {
	if c.replied {
		c.alreadyReplied()
		return
	}

	c.ResponseWriter.Header().Set("Content-Type", "application/json")
	json.NewEncoder(c.ResponseWriter).Encode(data)
}

func (c *Context) GetHeader(h string) string {
	return c.Request.Header.Get(h)
}

func (c *Context) alreadyReplied() {
	panic("already replied")
}

func (c *Context) Redirect(url string, status ...int) {
	if c.replied {
		c.alreadyReplied()
		return
	}
	code := 307
	if len(status) > 0 {
		code = status[0]
	}
	http.Redirect(c.ResponseWriter, c.Request, url, code)

	c.replied = true
}

func (c *Context) Render(data ...any) {

	mime, _ := accept.Negotiate(c.GetHeader("Accept"), "text/html", "application/json", "text/plain")

	panic(mime)
}

func (c *Context) GetParam(name string) string {
	return c.Request.FormValue(name)
}

func (c *Context) WriteString(data string) {
	c.Write([]byte(data))
}

func (c *Context) Write(data []byte) {
	c.ResponseWriter.Write(data)
}
