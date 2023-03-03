package component

import (
	"io"

	"golazy.dev/lazyview/script"
	"golazy.dev/lazyview/style"
)

type Local struct {
	Imports ImportMap
	Scripts []script.Script
	Styles  []style.Style
	Head    []io.WriterTo
}

// ImportMap returns the import map for the component.
func (l *Local) ImportMap() ImportMap {
	return l.Imports
}

// PageHead returns the head for the component.
func (l *Local) PageHead() []io.WriterTo {
	return l.Head
}

// PageStyles returns the styles for the component.
func (l *Local) PageStyles() []style.Style {
	return l.Styles
}

// PageScripts returns the scripts for the component.
func (l *Local) PageScripts() []script.Script {
	return l.Scripts
}
