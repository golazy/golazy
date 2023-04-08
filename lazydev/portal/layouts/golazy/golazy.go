package golazy

import (
	"bytes"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"portal/assets"
	"portal/components/nav"
	"strings"

	_ "embed"

	"golazy.dev/lazyaction"
	lazyassets "golazy.dev/lazyassets"
	"golazy.dev/lazyview/components/es_module_shims"
	"golazy.dev/lazyview/components/turbo"
	. "golazy.dev/lazyview/html"
)

//go:embed style.css
var css string

//go:embed slogans.txt
var slogansC string

var slogans = strings.Split(slogansC, "\n")

func init() {
	assets.Stylesheet.Add(css)
}

// Layout is the layout for the portal
type Layout struct {
	lazyaction.Base
}

func cite() string {
	return slogans[rand.Intn(len(slogans))]
}

func (l *Layout) RenderLayout(w http.ResponseWriter, current *url.URL, lass *lazyassets.Assets, content []byte) io.WriterTo {
	if l.SkipL {
		return bytes.NewBuffer(content)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "" && ct != "text/html" {
		return bytes.NewBuffer(content)
	}
	if lass == nil {
		panic("What!")
	}
	l.Page.Assets = lass
	l.Use(es_module_shims.Component)
	l.Assets = lass
	l.Charset = "utf-8"
	l.Title = "GoLazy" + " " + l.Title
	l.Viewport = "width=device-width, initial-scale=1"

	l.AddStylesheet(assets.Stylesheet)

	l.Content = Body(
		nav.Navigation(current),
		Main(

			content,
		),
	)
	l.Use(turbo.Component)

	return l.Element()
}
