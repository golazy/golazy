package jsoncode_test

import (
	"strings"
	"testing"

	"golazy.dev/lazycode"
	"golazy.dev/lazycode/jsoncode"
)

func TestDependencyEditIsStructuredAndIdempotent(t *testing.T) {
	source := []byte("{\n    \"name\": \"app\",\n    \"dependencies\": {\n        \"existing\": \"1.0.0\"\n    }\n}\n")
	workspace, err := lazycode.FromFiles("", map[string][]byte{"package.json": source})
	if err != nil {
		t.Fatal(err)
	}
	operation := jsoncode.Dependency("package.json", "dependencies", "@hotwired/turbo", "8.0.0")
	result, err := workspace.Plan(operation)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 1 {
		t.Fatalf("edits = %#v", result.Files)
	}
	after := string(result.Files[0].After)
	if !strings.Contains(after, `"@hotwired/turbo": "8.0.0"`) || !strings.Contains(after, `"existing": "1.0.0"`) {
		t.Fatalf("package.json =\n%s", after)
	}
	if !strings.Contains(after, "\n    \"dependencies\"") || !strings.HasSuffix(after, "\n") {
		t.Fatalf("indent or final newline changed: %q", after)
	}

	next, err := lazycode.FromFiles("", map[string][]byte{"package.json": result.Files[0].After})
	if err != nil {
		t.Fatal(err)
	}
	idempotent, err := next.Plan(operation)
	if err != nil {
		t.Fatal(err)
	}
	if idempotent.Changed() {
		t.Fatalf("second plan = %#v", idempotent.Files)
	}
}

func TestNestedSetAndRemove(t *testing.T) {
	document, err := jsoncode.Parse([]byte(`{"scripts":{"test":"go test ./..."}}`))
	if err != nil {
		t.Fatal(err)
	}
	changed, err := document.Set([]string{"scripts", "build"}, "lazy build")
	if err != nil || !changed {
		t.Fatalf("Set = %v, %v", changed, err)
	}
	changed, err = document.Remove([]string{"scripts", "test"})
	if err != nil || !changed {
		t.Fatalf("Remove = %v, %v", changed, err)
	}
	data, err := document.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), `"test"`) || !strings.Contains(string(data), `"build"`) {
		t.Fatalf("JSON = %s", data)
	}
}

func TestSetRejectsTraversalThroughScalar(t *testing.T) {
	document, err := jsoncode.Parse([]byte(`{"scripts":"not-an-object"}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := document.Set([]string{"scripts", "test"}, "command"); err == nil {
		t.Fatal("Set error = nil")
	}
	if _, err := document.EnsureDependency("scripts", "x", "1"); err == nil {
		t.Fatal("unsupported dependency group error = nil")
	}
	if _, err := jsoncode.Parse([]byte(`{} trailing`)); err == nil {
		t.Fatal("trailing invalid JSON error = nil")
	}
}
