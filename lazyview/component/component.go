package component

import (
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

type ComponentWithAssets interface {
	Assets() map[string]string
}

var allComponents = []Component{}

func Register(c Component) Component {
	allComponents = append(allComponents, c)
	return c
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
