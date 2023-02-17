package component

import (
	. "golazy.dev/lazyview/html"
	"golazy.dev/lazyview/nodes"
)

type Component []any

func (c Component) Header() []any {
	return []any(c)
}

var EsModuleShims = Component([]any{
	Script(Async(), nodes.NewAttr("nomodule"), Src("https://ga.jspm.io/npm:es-module-shims@1.4.6/dist/es-module-shims.js"), Crossorigin(("anonymous"))),
})

var Turbo = Component([]any{
	Script(Type("module"), Src("https://cdn.skypack.dev/@hotwired/turbo")),
})
