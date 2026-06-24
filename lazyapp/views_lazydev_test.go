//go:build lazydev

package lazyapp

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestOpenConfiguredViewsUsesLocalViewsInLazyDevBuild(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "views", "layouts", "app.html.tpl"), "local")

	previous := ViewsPath
	ViewsPath = filepath.Join(dir, "views")
	t.Cleanup(func() {
		ViewsPath = previous
	})

	views, err := openConfiguredViews(func() (fs.FS, error) {
		return fstest.MapFS{
			"layouts/app.html.tpl": {Data: []byte("embedded")},
		}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	content, err := fs.ReadFile(views, "layouts/app.html.tpl")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(content), "local"; got != want {
		t.Fatalf("layout = %q, want %q", got, want)
	}
}

func TestOpenConfiguredPublicUsesLocalPublicInLazyDevBuild(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "public", "styles.css"), "local")

	previous := PublicPath
	PublicPath = filepath.Join(dir, "public")
	t.Cleanup(func() {
		PublicPath = previous
	})

	public, err := openConfiguredPublic(func() (fs.FS, error) {
		return fstest.MapFS{
			"styles.css": {Data: []byte("embedded")},
		}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	content, err := fs.ReadFile(public, "styles.css")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(content), "local"; got != want {
		t.Fatalf("styles.css = %q, want %q", got, want)
	}
}

func writeFile(t *testing.T, filename string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
