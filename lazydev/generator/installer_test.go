package generator

import (
	"embed"
	"html/template"
	"os"
	"testing"
)

//go:embed sample
var SamplePrj embed.FS

var expectation = `name: Juan
prg: lazydev`

func TestInstaller(t *testing.T) {
	wd, _ := os.Getwd()
	os.RemoveAll(wd + "/test_out")
	defer os.RemoveAll(wd + "/test_out")

	i := &Project{
		FS:         SamplePrj,
		Dest:       wd + "/test_out",
		TrimPrefix: "sample",
		FuncMap: template.FuncMap{
			"name": func() string { return "Juan" },
		},
		Data: map[string]interface{}{
			"Prg": "lazydev",
		},
	}

	err := i.Install()
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(wd + "/test_out/folder/asdf.go")
	if err != nil {
		t.Fatal(err)
	}

	if string(data) != expectation {
		t.Fatalf("Expected %q, got %q", expectation, string(data))

	}

}
