package lazyapp

import (
	"github.com/spf13/cobra"
	"golazy.dev/lazyview/component"
)

var assetsCmd = func(a *App) *cobra.Command {

	return &cobra.Command{
		Use:   "assets",
		Short: "Install all the assets",
		Run: func(c *cobra.Command, args []string) {
			component.InstallAll(component.InstallOptions{
				Path:  "assets/public",
				Cache: "assets/cache",
			})

		},
	}

}
