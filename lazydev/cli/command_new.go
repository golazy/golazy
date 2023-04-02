package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
	"golazy.dev/lazydev/generator/base"

	_ "embed"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "new",
		Short: "Create a new lazy project",
		Args:  cobra.MatchAll(cobra.ExactArgs(1)),
		Run: func(cmd *cobra.Command, args []string) {

			wd, err := os.Getwd()
			if err != nil {
				wd = "."
			}

			target := filepath.Join(wd, args[0])

			err = base.Project.Generate(args[0], map[string]string{
				"Name": args[0],
			})
			if err != nil {
				panic(err)
			}
			shell := os.Getenv("SHELL")
			if shell == "" {
				return
			}
			fmt.Println(shell)
			subShell := exec.Command(shell)
			subShell.Dir = target
			subShell.Stdout = os.Stdout
			subShell.Stderr = os.Stderr
			subShell.Stdin = os.Stdin
			subShell.Run()

		},
	})
}
