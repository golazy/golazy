package lazylayout

import (
	_ "embed"
	"io"

	"github.com/golazy/golazy/lazyview/document"
	. "github.com/golazy/golazy/lazyview/html"
)

//go:embed style.css
var style string

var Layout = &document.Document{
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
