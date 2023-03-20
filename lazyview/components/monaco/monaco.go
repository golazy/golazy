package turbo

import (
	"golazy.dev/lazyview/component"
)

var Component = component.Register(&component.Npm{
	Name:    "monaco-editor",
	Version: "0.36.1",
})
