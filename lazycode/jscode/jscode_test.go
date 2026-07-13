package jscode_test

import (
	"strings"
	"testing"

	"golazy.dev/lazycode"
	"golazy.dev/lazycode/jscode"
)

func TestEnsureManagedBlockCreatesUpdatesAndIsIdempotent(t *testing.T) {
	source := []byte("console.log(\"app\");\n")
	created, changed, err := jscode.EnsureManagedBlock(source, "seo/config", "const seo = true;\n")
	if err != nil || !changed {
		t.Fatalf("create = %v, %v", changed, err)
	}
	if !strings.Contains(string(created), "// golazy:begin seo/config\nconst seo = true;\n// golazy:end seo/config") {
		t.Fatalf("created JS =\n%s", created)
	}
	idempotent, changed, err := jscode.EnsureManagedBlock(created, "seo/config", "const seo = true;\n")
	if err != nil || changed || string(idempotent) != string(created) {
		t.Fatalf("idempotent = %v, %v:\n%s", changed, err, idempotent)
	}
	updated, changed, err := jscode.EnsureManagedBlock(created, "seo/config", "const seo = false;")
	if err != nil || !changed || strings.Contains(string(updated), "true") {
		t.Fatalf("update = %v, %v:\n%s", changed, err, updated)
	}
	removed, changed, err := jscode.RemoveManagedBlock(updated, "seo/config")
	if err != nil || !changed || strings.Contains(string(removed), "golazy:begin") {
		t.Fatalf("remove = %v, %v:\n%s", changed, err, removed)
	}
}

func TestEnsureAndRemoveExactImport(t *testing.T) {
	source := []byte("import { Application } from \"@hotwired/stimulus\";\n\nApplication.start();\n")
	statement := `import "@hotwired/turbo";`
	result, changed, err := jscode.EnsureImport(source, statement)
	if err != nil || !changed {
		t.Fatalf("EnsureImport = %v, %v", changed, err)
	}
	if !strings.HasPrefix(string(result), "import { Application } from \"@hotwired/stimulus\";\nimport \"@hotwired/turbo\";") {
		t.Fatalf("JS =\n%s", result)
	}
	result, changed, err = jscode.RemoveImport(result, statement)
	if err != nil || !changed || strings.Contains(string(result), "@hotwired/turbo") {
		t.Fatalf("RemoveImport = %v, %v:\n%s", changed, err, result)
	}
}

func TestEnsureImportKeepsLicenseHeaderFirst(t *testing.T) {
	source := []byte("// Copyright GoLazy\n\nconsole.log(1);\n")
	result, changed, err := jscode.EnsureSideEffectImport(source, "addon")
	if err != nil || !changed {
		t.Fatalf("EnsureSideEffectImport = %v, %v", changed, err)
	}
	if !strings.HasPrefix(string(result), "// Copyright GoLazy\n\nimport \"addon\";") {
		t.Fatalf("JS =\n%s", result)
	}
}

func TestOperationsComposeInWorkspace(t *testing.T) {
	workspace, err := lazycode.FromFiles("", map[string][]byte{"app/js/app.js": []byte("console.log(1);\n")})
	if err != nil {
		t.Fatal(err)
	}
	result, err := workspace.Plan(
		jscode.Import("app/js/app.js", `import "@hotwired/turbo";`),
		jscode.ManagedBlock("app/js/app.js", "addon", "Turbo.start();"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 1 || !strings.Contains(string(result.Files[0].After), "Turbo.start();") {
		t.Fatalf("result = %#v", result.Files)
	}
}

func TestManagedBlocksRejectMalformedMarkers(t *testing.T) {
	source := []byte("// golazy:begin x\ncode();\n")
	if _, _, err := jscode.EnsureManagedBlock(source, "x", "newCode();"); err == nil {
		t.Fatal("malformed marker error = nil")
	}
	if _, _, err := jscode.EnsureImport(source, "const x = 1;"); err == nil {
		t.Fatal("invalid import error = nil")
	}
}
