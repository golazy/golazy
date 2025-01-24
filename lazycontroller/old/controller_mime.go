package lazycontroller

import (
	"mime"
	"strings"
)

func (c *Controller) Accepts() Accepts {
	a := make(Accepts)
	for _, v := range strings.Split(c.Request.Header.Get("Accept"), ",") {
		t, p, err := mime.ParseMediaType(v)
		if err != nil {
			continue
		}
		a[MimeType(t)] = MimeTypeParams(p)
	}
	return a
}

func (c *Controller) WantsTurbo() bool {
	return c.Wants("turbo")
}

func (c *Controller) WantsHTML() bool {
	return c.Wants("html")
}
func (c *Controller) WantsJSON() bool {
	return c.Wants("json")
}

func (c *Controller) Wants(mimes ...string) bool {
	accepts := c.Accepts()
	for _, v := range mimes {
		switch v {
		case "html":
			v = "text/html"
		case "json":
			v = "application/json"
		case "text", "plain", "txt":
			v = "text/plain"
		case "csv":
			v = "text/csv"
		case "turbo":
			v = "text/vnd.turbo-stream.html"
		case "*":
			v = "*/*"
		}

		if _, ok := accepts[MimeType(v)]; ok {
			return true
		}
	}
	return false
}

type MimeType string
type MimeTypeParams map[string]string
type Accepts map[MimeType]MimeTypeParams
