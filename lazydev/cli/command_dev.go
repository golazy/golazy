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

			interrupt := make(chan os.Signal, 1)
			signal.Notify(interrupt, os.Interrupt)

			// Launch normal devserver if noportal is set
			if np {
				srv := devserver.New(devserver.Options{
					BuildDir:  args[0],
					RootDir:   findFirstRoot(mainDir),
					BuildArgs: strings.Split("-buildvcs=false", " "),
					RunEnv:    []string{"PORT=" + port},
					RunArgs:   SubArgs,
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
				Addr:      port,
				BuildDir:  args[0],
				BuildArgs: strings.Split("-buildvcs=false", " "),
				RunArgs:   SubArgs,
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
