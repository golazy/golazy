package lazycontroller

import (
	"mime"
	"strings"
)

func (c *Base) Accepts() Accepts {
	a := make(Accepts)
	for _, v := range strings.Split(c.R.Header.Get("Accept"), ",") {
		t, p, err := mime.ParseMediaType(v)
		if err != nil {
			continue
		}
		a[MimeType(t)] = MimeTypeParams(p)
	}
	return a
}

func (c *Base) WantsTurbo() bool {
	return c.Wants("turbo")
}

func (c *Base) WantsHTML() bool {
	return c.Wants("html")
}
func (c *Base) WantsJSON() bool {
	return c.Wants("json")
}

func (c *Base) Wants(mimes ...string) bool {
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
