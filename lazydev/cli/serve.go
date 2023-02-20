package cli

import (
	"fmt"
	"go/build"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"golazy.dev/lazydev/server"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "serve [main package dir]",
		Short: "Start development server",
		Run: func(cmd *cobra.Command, args []string) {

			pkg, err := build.ImportDir(args[0], build.IgnoreVendor)
			if err != nil || pkg.Name != "main" {
				fmt.Println("Error:", args[0], "is not a main package")
				os.Exit(-1)
				return
			}

			s := &server.Server{
				BuildDir:        args[0],
				BuildArgs:       strings.Split("-buildvcs=false", " "),
				HttpHandler:     StringHandler("http"),
				PrefixHandler:   StringHandler("golazy"),
				FallbackHandler: StringHandler("fallback"),
			}

			err = s.ListenAndServe()
			if err == nil || err == http.ErrServerClosed {
				return
			}

		},
		Args: cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),

		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return mainPackages(), cobra.ShellCompDirectiveNoFileComp
		},
	})
}

type StringHandler string

func (h StringHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(h))
}

func mainPackages() []string {
	var mains []string
	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}

	moduleRoot := findRoot(wd)

	filepath.WalkDir(moduleRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			return nil
		}

		pack, err := build.ImportDir(path, build.IgnoreVendor)
		if err != nil {
			return nil
		}
		if pack.Name != "main" {
			return nil
		}

		rel, err := filepath.Rel(wd, path)
		if err != nil {
			mains = append(mains, path)
		}
		mains = append(mains, rel)
		return nil
	})
	return mains

}

func findRoot(wd string) string {
	root := wd
	for current := root; current != filepath.Dir(current); current = filepath.Dir(current) {
		info, err := os.Stat(filepath.Join(current, "go.mod"))
		if err != nil {
			continue
		}
		if !info.IsDir() {
			root = current
		}
	}
	return root
}
