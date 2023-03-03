package component

import (
	"golazy.dev/lazyview/script"
	"golazy.dev/lazyview/style"
)

type Git struct {
	CloneURL string
	Tag      string
	Imports  ImportMap
	Fiels    []string
	Scripts  []script.Script
	Styles   []style.Style
}

func (g *Git) Install(opts InstallOptions) error {
	panic("not implemented")
}

func (g *Git) Uninstall(opts InstallOptions) error {
	panic("not implemented")
}
