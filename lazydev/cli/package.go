package cli

import (
	"fmt"
	"go/build"
	"io/fs"
	"os"
	"path/filepath"
)

func findFirstRoot(wd string) string {
	root := wd
	for current := root; current != filepath.Dir(current); current = filepath.Dir(current) {
		info, err := os.Stat(filepath.Join(current, "go.mod"))
		if err == nil && !info.IsDir() {
			return current
		}
	}
	return wd
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
	fmt.Println(mains)
	return mains

}
