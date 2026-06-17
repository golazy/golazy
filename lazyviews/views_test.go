package lazyviews

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveFindsRelativeViewsPathFromParentDirectory(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "app", "views", "layouts", "app.html.tpl"), "layout")
	child := filepath.Join(root, "cmd", "app")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(child)

	dir, err := Resolve("app/views")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := dir, filepath.Join(root, "app", "views"); got != want {
		t.Fatalf("Resolve() = %q, want %q", got, want)
	}
}

func TestResolveRejectsMissingLayout(t *testing.T) {
	_, err := Resolve(filepath.Join(t.TempDir(), "views"))
	if err == nil {
		t.Fatal("Resolve() error is nil")
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
