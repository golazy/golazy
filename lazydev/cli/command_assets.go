package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"golazy.dev/lazydev/build"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "assets [main package dir]",
		Short: "Install all assets",
		Run: func(cmd *cobra.Command, args []string) {

			root := findFirstRoot(args[0])

			f, err := os.CreateTemp("", "lazydev-build-*")
			if err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(-1)
				return
			}
			path := f.Name()
			f.Close()

			err = build.Build(build.Options{
				Dir:        args[0],
				Args:       []string{"-buildvcs=false"},
				OutputPath: path,
			})

			if err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(-1)
				return
			}

			appCmd := exec.Command(path, "assets")
			appCmd.Dir = root
			appCmd.Stdout = os.Stdout
			appCmd.Stderr = os.Stderr
			err = appCmd.Run()
			if err != nil {
				os.Exit(appCmd.ProcessState.ExitCode())
			}
		},
		Args: cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return mainPackages(), cobra.ShellCompDirectiveNoFileComp
		},
	})
}
