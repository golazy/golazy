package lazydoc

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDirRecordsSourcePositions(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.test/sample\n\ngo 1.26.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	source := `// Package sample documents source positions.
package sample

const Answer = 42

var Name = "sample"

func Run() {}

type Runner struct{}

func (Runner) Start() {}
`
	if err := os.WriteFile(filepath.Join(dir, "doc.go"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	index, err := LoadDir(dir, "local")
	if err != nil {
		t.Fatal(err)
	}
	version, ok := index.Version("local")
	if !ok {
		t.Fatal("local version missing")
	}
	pkg, ok := version.Package("example.test/sample")
	if !ok {
		t.Fatal("package missing")
	}

	assertSource(t, pkg.Source, "doc.go", 1)
	assertSource(t, pkg.Constants[0].Source, "doc.go", 4)
	assertSource(t, pkg.Variables[0].Source, "doc.go", 6)
	assertSource(t, pkg.Functions[0].Source, "doc.go", 8)
	assertSource(t, pkg.Types[0].Source, "doc.go", 10)
	assertSource(t, pkg.Types[0].Methods[0].Source, "doc.go", 12)
}

func assertSource(t *testing.T, got *Source, file string, line int) {
	t.Helper()
	if got == nil {
		t.Fatalf("source is nil, want %s:%d", file, line)
	}
	if got.File != file || got.Line != line {
		t.Fatalf("source = %#v, want %s:%d", got, file, line)
	}
}
