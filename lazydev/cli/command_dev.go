package cli

import (
	"fmt"
	"go/build"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golazy.dev/lazydev/cli/commands/dev"
)

func init() {
	cmd := &cobra.Command{
		Use:   "dev [flags] [main package dir] -- [program args]",
		Short: "Start development server",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 1 {
				panic("No main package dir specified")
			}

			mainDir, err := filepath.Abs(args[0])
			if err != nil {
				panic(err)
			}

			// Check the package is a main package
			pkg, err := build.ImportDir(mainDir, build.IgnoreVendor)
			if err != nil || pkg.Name != "main" {
				fmt.Println("Error:", args[0], "is not a main package")
				os.Exit(-1)
				return
			}

			np, err := cmd.Flags().GetBool("noportal")
			if err != nil {
				panic(err)
			}

			port, err := cmd.Flags().GetString("port")
			if err != nil {
				panic(err)
			}

			dev.Run(dev.DevOpts{
				MainDir: mainDir,
				Dir:     findFirstRoot(mainDir),
				Portal:  !np,
				Port:    port,
			})

		},
		Args: cobra.MatchAll(cobra.MinimumNArgs(1), cobra.OnlyValidArgs),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return mainPackages(), cobra.ShellCompDirectiveNoFileComp
		},
	}
	cmd.PersistentFlags().BoolP("noportal", "n", false, "Disable portal and serve only the main package")
	viper.BindPFlag("noportal", cmd.PersistentFlags().Lookup("nosportal"))
	viper.BindEnv("noportal")

	cmd.PersistentFlags().StringP("port", "p", "127.0.0.1:2000", "Listen port (or address:port pair)")
	viper.BindPFlag("port", cmd.PersistentFlags().Lookup("port"))
	viper.BindEnv("port")

	rootCmd.AddCommand(cmd)

}

func includes(list []string, item string) bool {
	for _, i := range list {
		if i == item {
			return true
		}
	}
	return false
}
