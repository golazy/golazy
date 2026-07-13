//go:build lazydev

package lazyapp_test

import (
	"io/fs"
	"os"
	"path/filepath"

	"golazy.dev/lazyapp"
)

func configureExampleFiles(public, views fs.FS) func() {
	root, err := os.MkdirTemp("", "golazy-lazyapp-example-*")
	if err != nil {
		panic(err)
	}
	copyExampleFS(filepath.Join(root, "app/public"), public)
	copyExampleFS(filepath.Join(root, "app/views"), views)
	previousPublic, previousViews := lazyapp.PublicPath, lazyapp.ViewsPath
	lazyapp.PublicPath = filepath.Join(root, "app/public")
	lazyapp.ViewsPath = filepath.Join(root, "app/views")
	return func() {
		lazyapp.PublicPath = previousPublic
		lazyapp.ViewsPath = previousViews
		_ = os.RemoveAll(root)
	}
}

func copyExampleFS(target string, source fs.FS) {
	if err := fs.WalkDir(source, ".", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if name == "." {
			return os.MkdirAll(target, 0o755)
		}
		path := filepath.Join(target, filepath.FromSlash(name))
		if entry.IsDir() {
			return os.MkdirAll(path, 0o755)
		}
		data, err := fs.ReadFile(source, name)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		return os.WriteFile(path, data, 0o644)
	}); err != nil {
		panic(err)
	}
}
