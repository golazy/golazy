package cli

import (
	"fmt"
	"go/build"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"portal/apps/portal"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golazy.dev/lazydev/devserver"
	"golazy.dev/lazydev/devserver/events"
	"golazy.dev/lazydev/portalserver"
)

func init() {
	cmd := &cobra.Command{
		Use:   "dev [main package dir]",
		Short: "Start development server",
		Run: func(cmd *cobra.Command, args []string) {

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

			interrupt := make(chan os.Signal, 1)
			signal.Notify(interrupt, os.Interrupt)

			// Launch normal devserver if noportal is set
			if np {
				srv := devserver.New(devserver.Options{
					BuildDir:  args[0],
					RootDir:   findFirstRoot(mainDir),
					BuildArgs: strings.Split("-buildvcs=false", " "),
					RunEnv:    []string{"PORT=2000"},
					Events: func(e events.Event) {

						if e, ok := e.(events.Stdout); ok {
							os.Stdout.Write([]byte(e))
							return
						}

						if e, ok := e.(events.Stderr); ok {
							os.Stderr.Write([]byte(e))
							return
						}

						if e, ok := e.(events.BuildError); ok {
							fmt.Println(string(e.Out))
							return
						}

						fmt.Printf("#> %-15s %s\n", e.Type(), e.String())
					},
				})
				go func() {
					<-interrupt
					fmt.Println("Got CTRL+C, shutting down...")
					srv.Close()
				}()
				err := srv.Serve()
				if err != nil {
					panic(err)
				}
				return
			}

			srv := portalserver.New(portalserver.Options{
				Addr:      "127.0.0.1:2000",
				BuildDir:  args[0],
				BuildArgs: strings.Split("-buildvcs=false", " "),
				App:       portal.App,
			})

			go func() {
				<-interrupt
				fmt.Println("Got CTRL+C, shutting down...")
				srv.Close()
			}()

			srv.ListenAndServe()
			if err != nil {
				panic(err)
			}

		},
		Args: cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return mainPackages(), cobra.ShellCompDirectiveNoFileComp
		},
	}
	cmd.PersistentFlags().BoolP("noportal", "n", false, "Disable portal and serve only the main package")
	viper.BindPFlag("noportal", cmd.PersistentFlags().Lookup("nosportal"))
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
