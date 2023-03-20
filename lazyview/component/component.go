package component

import (
	"bytes"
	"fmt"
	"io"

	"golazy.dev/lazyview/script"
	"golazy.dev/lazyview/style"
)

type Component interface {
	String() string
}

type ComponentWithImports interface {
	ImportMap() map[string]string
}
type ComponentWithHead interface {
	PageHead() []io.WriterTo
}
type ComponentWithStyles interface {
	PageStyles() []style.Style
}
type ComponentWithScripts interface {
	PageScripts() []script.Script
}
type ComponentWithMaps interface {
	ImportMap() ImportMap
}
type ComponentWithInstall interface {
	Install(opts InstallOptions) error
	Uninstall(opts InstallOptions) error
	Installed(opts InstallOptions) bool
}

type ComponentWithUninstall interface {
}

type InstallOptions struct {
	Path, Cache string
}

var DefaultInstallOptions = InstallOptions{}

type ComponentWithAssets interface {
	Assets() map[string]string
}

var allComponents = []Component{}

type ComponentState struct {
	Name      string
	Imports   map[string]string
	Head      string
	Styles    string
	Scripts   string
	Maps      map[string]string
	Installed bool
}

func All() []ComponentState {
	cs := make([]ComponentState, len(allComponents))
	for i, c := range allComponents {
		s := ComponentState{
			Name: c.String(),
		}
		if ci, ok := c.(ComponentWithImports); ok {
			s.Imports = ci.ImportMap()
		}
		if ci, ok := c.(ComponentWithHead); ok {
			buf := new(bytes.Buffer)
			for _, h := range ci.PageHead() {
				h.WriteTo(buf)
			}
			s.Head = buf.String()
		}
		if ci, ok := c.(ComponentWithStyles); ok {
			buf := new(bytes.Buffer)
			for _, h := range ci.PageStyles() {
				h.WriteTo(buf)
			}
			s.Styles = buf.String()
		}
		if ci, ok := c.(ComponentWithScripts); ok {
			buf := new(bytes.Buffer)
			for _, h := range ci.PageScripts() {
				h.WriteTo(buf)
			}
			s.Scripts = buf.String()
		}

		if ci, ok := c.(ComponentWithMaps); ok {
			s.Maps = ci.ImportMap()
		}
		if ci, ok := c.(ComponentWithInstall); ok {
			s.Installed = ci.Installed(DefaultInstallOptions)
		}

		cs[i] = s
	}
	return cs
}

func Register(c Component) Component {
	allComponents = append(allComponents, c)
	return c
}

func Find(name string) Component {
	for _, c := range allComponents {
		if c.String() == name {
			return c
		}
	}
	return nil
}

func InstallAll(opts InstallOptions) error {
	fmt.Println("Installing components...", allComponents)
	for _, c := range allComponents {
		if ci, ok := c.(ComponentWithInstall); ok {

			if !ci.Installed(opts) {
				fmt.Println("Missing!")
				fmt.Println("Installing", c.String(), "with options", opts)
				err := ci.Install(opts)
				if err != nil {
					fmt.Println(c.String(), "install error:", err)
					return err
				}
			}
			fmt.Println("OK")
		}
	}
	return nil
}
