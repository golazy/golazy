package generator

import (
	"embed"
	"html/template"
	"os"
	"strings"
	"testing"
)

type TestData struct {
	Name string
}

func (t *TestData) FirstName() string {
	parts := strings.Split(t.Name, " ")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

func TestPathTemplate(t *testing.T) {

	a := &FSGenerator{
		data: &TestData{Name: "Guillermo Alvarez"},
		FuncMap: template.FuncMap{
			"folder": func() string {
				return "folder"
			},
		},
	}

	test := func(input, expected string) {
		t.Helper()
		if a.pathTemplate(input) != expected {
			t.Fatalf("pathTemplate failed for %s. Expected %q. Got %q",
				input,
				expected,
				a.pathTemplate(input))
		}
	}

	test("{{.Name}}", "Guillermo Alvarez")
	test("{{.FirstName}}", "Guillermo")
	test("{{folder}}", "folder")

}

//go:embed sample
var SamplePrj embed.FS

var expectation = `name: Juan
prg: lazy_dev`

func TestInstaller(t *testing.T) {
	wd, _ := os.Getwd()
	os.RemoveAll(wd + "/test_out")
	defer os.RemoveAll(wd + "/test_out")

	i := &FSGenerator{
		FS:         SamplePrj,
		dest:       wd + "/test_out",
		TrimPrefix: "sample",
		FuncMap: template.FuncMap{
			"name":   func() string { return "Juan" },
			"folder": func() string { return "folder" },
		},
	}

	err := i.Generate(wd+"/test_out", map[string]string{
		"App": "Lazy Dev",
	})
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
