package es_module_shims

import (
	"golazy.dev/lazyview/component"
)

var Component = component.Register(&component.URL{
	URL:  "https://ga.jspm.io/npm:es-module-shims@1.7.0/dist/es-module-shims.js",
	Path: "/es-module-shims.js",
})
