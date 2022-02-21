package layout

import (
	. "github.com/guillermo/golazy/lazyview/html"
	"github.com/guillermo/golazy/lazyview/nodes"
)

var BasicLayout = &LayoutTemplate{
	Lang:     "en",
	Title:    "lazyview",
	Viewport: "width=device-width",
	Styles:   []string{SimpleCSS(), PageStyle()},
	Head: []interface{}{
		Script(Async(), Src("https://ga.jspm.io/npm:es-module-shims@1.4.6/dist/es-module-shims.js"), Crossorigin(("anonymous"))),
		Script(Type("module"),
			nodes.Raw(`import hotwiredTurbo from 'https://cdn.skypack.dev/@hotwired/turbo';`),
		),
	},
	LayoutBody: LayoutBody,
}
