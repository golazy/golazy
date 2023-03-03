package cli

import (
	"fmt"
	"go/build"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golazy.dev/lazydev/internal/portal/apps/portal"
	"golazy.dev/lazydev/server"
)

func init() {
	cmd := &cobra.Command{
		Use:   "serve [main package dir]",
		Short: "Start development server",
		Run: func(cmd *cobra.Command, args []string) {

			pkg, err := build.ImportDir(args[0], build.IgnoreVendor)
			if err != nil || pkg.Name != "main" {
				fmt.Println("Error:", args[0], "is not a main package")
				os.Exit(-1)
				return
			}

			s := server.New(server.Options{
				BuildDir:  args[0],
				BuildArgs: strings.Split("-buildvcs=false", " "),
				App:       portal.Portal,
			})

			err = s.ListenAndServe()

			if err == nil || err == http.ErrServerClosed {
				return
			}

		},
		Args: cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return mainPackages(), cobra.ShellCompDirectiveNoFileComp
		},
	}
	cmd.PersistentFlags().BoolP("nosportal", "n", false, "Disable portal and serve only the main package")
	viper.BindPFlag("nosportal", cmd.PersistentFlags().Lookup("nosportal"))
	viper.BindEnv("noportal")

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
