package lazycontroller

import (
	"os"
	"path/filepath"
	"testing"
)

func TestErrorPathFormatterUsesWorkspaceRoot(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "sample_app", "app", "controllers", "home_controller", "homecontroller.go")

	formatter := errorPathFormatter{roots: []string{root}}

	if got, want := formatter.displayFile(file), "sample_app/app/controllers/home_controller/homecontroller.go"; got != want {
		t.Fatalf("displayFile() = %q, want %q", got, want)
	}
}

func TestErrorPathFormatterUsesCurrentDirectoryRoot(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "app", "controllers", "home_controller", "homecontroller.go")

	formatter := errorPathFormatter{roots: []string{root}}

	if got, want := formatter.displayFile(file), "app/controllers/home_controller/homecontroller.go"; got != want {
		t.Fatalf("displayFile() = %q, want %q", got, want)
	}
}

func TestErrorPathFormatterUsesModuleCachePath(t *testing.T) {
	file := filepath.Join(
		string(filepath.Separator),
		"home",
		"guillermo",
		"go",
		"pkg",
		"mod",
		"golang.org",
		"toolchain@v0.0.1-go1.26.2.linux-amd64",
		"src",
		"net",
		"http",
		"server.go",
	)

	formatter := errorPathFormatter{}

	if got, want := formatter.displayFile(file), "golang.org/toolchain@v0.0.1-go1.26.2.linux-amd64/src/net/http/server.go"; got != want {
		t.Fatalf("displayFile() = %q, want %q", got, want)
	}
}

func TestErrorPathFormatterUsesModulePath(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/app\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, "internal", "worker", "worker.go")
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("package worker\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	formatter := errorPathFormatter{}

	if got, want := formatter.displayFile(file), "example.com/app/internal/worker/worker.go"; got != want {
		t.Fatalf("displayFile() = %q, want %q", got, want)
	}
}
