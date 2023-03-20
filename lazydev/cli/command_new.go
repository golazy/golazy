package cli

import (
	"embed"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
	"golazy.dev/lazydev/cli/prjtmpl"

	_ "embed"
)

//go:embed all:project
var projectTemplate embed.FS

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

			p := prjtmpl.Project{
				FS:         projectTemplate,
				TrimPrefix: "project",
				Dest:       target,
				Data: map[string]any{
					"Name": args[0],
				},
			}

			err = p.Install()
			if err != nil {
				panic(err)
			}
			shell := os.Getenv("SHELL")
			if shell == "" {
				return
			}
			subShell := exec.Command(shell)
			subShell.Dir = target
			subShell.Stdout = os.Stdout
			subShell.Stderr = os.Stderr
			subShell.Stdin = os.Stdin
			subShell.Run()

		},
	})
}
