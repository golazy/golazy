package lazyapp

import "github.com/spf13/cobra"

var rootCmd = func(a *App) *cobra.Command {

	root := serveCmd(a)
	root.AddCommand(routesCmd(a))
	root.AddCommand(assetsCmd(a))

	return root
}
