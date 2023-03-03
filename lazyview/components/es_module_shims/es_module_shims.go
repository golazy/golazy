package es_module_shims

import (
	"golazy.dev/lazyview/component"
	"golazy.dev/lazyview/script"
)

var Component = component.Register(component.Local{
	Scripts: []script.Script{
		{
			Src:      "https://ga.jspm.io/npm:es-module-shims@1.6.3/dist/es-module-shims.js",
			NoModule: true,
			Async:    true,
		},
	},
})
