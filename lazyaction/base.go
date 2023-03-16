package lazyaction

import (
	"bytes"
	"io"
	"net/http"

	"golazy.dev/lazyassets"
	"golazy.dev/lazyview/components/turbo"
	"golazy.dev/lazyview/html"
	"golazy.dev/lazyview/page"
)

type setVars interface {
	setVars(http.ResponseWriter, *http.Request, *Action, *lazyassets.Assets)
}

type Base struct {
	w http.ResponseWriter
	r *http.Request
	a *Action
	page.Page
	SkipL bool
}

func (b *Base) setVars(w http.ResponseWriter, r *http.Request, a *Action, assets *lazyassets.Assets) {
	b.w = w
	b.r = r
	b.a = a
	b.Page.Assets = assets
}

func (l *Base) SkipLayout() {
	l.SkipL = true
}

func (l *Base) RenderLayout(w http.ResponseWriter, assets *lazyassets.Assets, content []byte) io.WriterTo {
	if l.SkipL {
		return bytes.NewBuffer(content)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "" && ct != "text/html" {
		return bytes.NewBuffer(content)
	}

	l.Page = page.Page{}
	l.Assets = assets
	l.Charset = "utf-8"
	l.Viewport = "width=device-width, initial-scale=1"
	l.Content = html.Body(content)
	l.Use(turbo.Component)

	return l.Element()
}
