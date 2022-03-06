package lazylayout

import (
	_ "embed"
	"io"

	. "github.com/guillermo/golazy/lazyview/html"
	"github.com/guillermo/golazy/lazyview/layout"
)

//go:embed style.css
var style string

var Layout = &layout.LayoutTemplate{
	Lang:     "en",
	Title:    "golazy",
	Viewport: "width=device-width",
	Styles:   []string{style},
	Head:     []interface{}{Script(Type("module"), Src("https://cdn.skypack.dev/@hotwired/turbo"))},
}

func PageHeader() io.WriterTo {
	return Header(H1("lazygo"))
}

func PageNav() io.WriterTo {
	return Nav(
		Ul(
			Li(A(Href("#"), "Web")),
			Li(A(Href("#"), "Docs")),
			Li(A(Href("#"), "Code")),
			Li(A(Href("#"), "Config")),
		),
	)
}
