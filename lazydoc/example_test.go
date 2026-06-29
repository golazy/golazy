package lazydoc_test

import (
	"fmt"
	"os"
	"path/filepath"

	"golazy.dev/lazydoc"
)

func ExampleLoadDir() {
	dir, err := os.MkdirTemp("", "lazydoc-example-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	err = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.test/sample\n\ngo 1.26.0\n"), 0o644)
	if err != nil {
		panic(err)
	}

	source := `// Package sample has documentation to index.
package sample

// Run starts the sample.
func Run() {}
`
	err = os.WriteFile(filepath.Join(dir, "doc.go"), []byte(source), 0o644)
	if err != nil {
		panic(err)
	}

	index, err := lazydoc.LoadDir(dir, "local")
	if err != nil {
		panic(err)
	}

	version, _ := index.Version("local")
	pkg, _ := version.Package("example.test/sample")
	results := index.Search("local", "run")

	fmt.Println(version.Module)
	fmt.Println(pkg.Synopsis)
	fmt.Println(results[0].URL)
	// Output:
	// example.test/sample
	// Package sample has documentation to index.
	// /packages/local/example.test.sample#Run
}
