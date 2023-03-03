package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"golazy.dev/lazydev/build"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "build [main package dir]",
		Short: "Build a lazydev app",
		Run: func(cmd *cobra.Command, args []string) {

			name := filepath.Base(args[0])
			fmt.Println("name:", name)

			arg, err := filepath.Abs(args[0])
			if err != nil {
				panic(err)
			}
			fmt.Println("arg:", arg)

			root := findFirstRoot(arg)
			fmt.Println("root:", root)
			output := filepath.Join(root, "build", name)
			fmt.Println("output:", output)

			fmt.Printf("cd %s && go build\n", args[0])
			err = build.Build(build.Options{
				Dir:        args[0],
				Args:       []string{"-buildvcs=false"},
				OutputPath: output,
			})
			if err != nil {
				fmt.Println("error:", err)
				os.Exit(-1)
				return
			}

		},
		Args: cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return mainPackages(), cobra.ShellCompDirectiveNoFileComp
		},
	})

}
