package gocode_test

import (
	"go/ast"
	"go/token"
	"strings"
	"testing"

	"golazy.dev/lazycode"
	"golazy.dev/lazycode/gocode"
)

func TestRewritePlansFormattedImportChanges(t *testing.T) {
	workspace, err := lazycode.FromFiles("", map[string][]byte{
		"app/init.go": []byte("package app\n\nimport \"context\"\n\nfunc F(_ context.Context) {}\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := workspace.Plan(gocode.Rewrite("app/init.go", func(_ *token.FileSet, file *ast.File) (bool, error) {
		changed := gocode.EnsureImport(file, "golazy.dev/lazydeps")
		if !gocode.UsesSelector(file, "context") {
			changed = gocode.RemoveImport(file, "context") || changed
		}
		return changed, nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 1 {
		t.Fatalf("edits = %#v", result.Files)
	}
	after := string(result.Files[0].After)
	if !strings.Contains(after, `"golazy.dev/lazydeps"`) || !strings.Contains(after, `"context"`) {
		t.Fatalf("rewritten Go =\n%s", after)
	}
}

func TestImportHelpersAreIdempotentAndSeeNewImports(t *testing.T) {
	_, file, err := gocode.Parse("file.go", []byte("package p\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !gocode.EnsureImport(file, "example.com/normal") {
		t.Fatal("EnsureImport changed = false")
	}
	if !gocode.HasImport(file, "example.com/normal") {
		t.Fatal("new import not visible")
	}
	if gocode.EnsureImport(file, "example.com/normal") {
		t.Fatal("second EnsureImport changed = true")
	}
	if changed, err := gocode.EnsureBlankImport(file, "example.com/addon"); err != nil || !changed {
		t.Fatalf("EnsureBlankImport = %v, %v", changed, err)
	}
	if changed, err := gocode.EnsureNamedImport(file, "addon", "example.com/addon"); err == nil || changed {
		t.Fatalf("conflicting EnsureNamedImport = %v, %v", changed, err)
	}
	if !gocode.RemoveImport(file, "example.com/normal") || gocode.HasImport(file, "example.com/normal") {
		t.Fatal("RemoveImport did not remove import")
	}
}

func TestRewriteSourceHonorsNoChangeAndRejectsInvalidGo(t *testing.T) {
	source := []byte("package p\n")
	result, changed, err := gocode.RewriteSource("file.go", source, func(*token.FileSet, *ast.File) (bool, error) {
		return false, nil
	})
	if err != nil || changed || string(result) != string(source) {
		t.Fatalf("RewriteSource = %q, %v, %v", result, changed, err)
	}
	if _, _, err := gocode.RewriteSource("bad.go", []byte("package"), func(*token.FileSet, *ast.File) (bool, error) {
		return false, nil
	}); err == nil {
		t.Fatal("invalid Go error = nil")
	}
}

func TestEnsureFileFormatsGeneratedSidecar(t *testing.T) {
	workspace := lazycode.New("")
	result, err := workspace.Plan(gocode.EnsureFile("app/controllers/seo_addon.go", []byte("package controllers\nfunc (c *BaseController)SEO(){}\n")))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 1 || result.Files[0].Kind != lazycode.EditCreate {
		t.Fatalf("result = %#v", result)
	}
	if !strings.Contains(string(result.Files[0].After), "func (c *BaseController) SEO()") {
		t.Fatalf("generated file =\n%s", result.Files[0].After)
	}
}

func TestImportNameReportsNormalAndBlankImports(t *testing.T) {
	_, file, err := gocode.Parse("imports.go", []byte("package imports\nimport (\n\t\"fmt\"\n\t_ \"example.com/runtime\"\n)\n"))
	if err != nil {
		t.Fatal(err)
	}
	if name, found, err := gocode.ImportName(file, "fmt"); err != nil || !found || name != "" {
		t.Fatalf("fmt import = %q, %t, %v", name, found, err)
	}
	if name, found, err := gocode.ImportName(file, "example.com/runtime"); err != nil || !found || name != "_" {
		t.Fatalf("runtime import = %q, %t, %v", name, found, err)
	}
}
