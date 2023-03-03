package component

import (
	"golazy.dev/lazyview/script"
	"golazy.dev/lazyview/style"
)

type Github struct {
	Repo    string
	Tag     string
	Imports ImportMap
	Files   []string
	Scripts []script.Script
	Styles  []style.Style
}
