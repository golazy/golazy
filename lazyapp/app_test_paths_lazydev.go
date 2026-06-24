//go:build lazydev

package lazyapp

import (
	"os"
	"path/filepath"
	"testing"
)

func configureLazyDevViewsForTest(t *testing.T, files map[string]string) {
	t.Helper()
	dir := t.TempDir()
	writeLazyDevTestFiles(t, dir, files)

	previous := ViewsPath
	ViewsPath = dir
	t.Cleanup(func() {
		ViewsPath = previous
	})
}

func configureLazyDevPublicForTest(t *testing.T, files map[string]string) {
	t.Helper()
	dir := t.TempDir()
	writeLazyDevTestFiles(t, dir, files)

	previous := PublicPath
	PublicPath = dir
	t.Cleanup(func() {
		PublicPath = previous
	})
}

func lazyDevTestBuild() bool {
	return true
}

func writeLazyDevTestFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		filename := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filename, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
