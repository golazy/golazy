package tomlcode_test

import (
	"strings"
	"testing"

	"golazy.dev/lazycode"
	"golazy.dev/lazycode/tomlcode"
)

func TestDocumentPreservesCommentsAndUnrelatedFormatting(t *testing.T) {
	source := []byte("# project\r\nschema = 1\r\n\r\n[addons.postgres] # keep table\r\nversion   = \"v1.0.0\"   # pinned\r\ndirect = true\r\n")
	document, err := tomlcode.Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := document.SetString("addons.postgres", "version", "v1.2.0")
	if err != nil || !changed {
		t.Fatalf("SetString = %v, %v", changed, err)
	}
	result := string(document.Bytes())
	if !strings.Contains(result, "version   = \"v1.2.0\"   # pinned") {
		t.Fatalf("updated TOML =\n%s", result)
	}
	if !strings.Contains(result, "[addons.postgres] # keep table") || !strings.Contains(result, "\r\n") {
		t.Fatalf("comments or line endings lost:\n%q", result)
	}
}

func TestSetCreatesTableAndIsIdempotent(t *testing.T) {
	workspace, err := lazycode.FromFiles("", map[string][]byte{"addons.toml": []byte("schema = 1\n")})
	if err != nil {
		t.Fatal(err)
	}
	operation := tomlcode.Set("addons.toml", `addons."postgres/jobs"`, "version", `"v1.2.0"`)
	result, err := workspace.Plan(operation)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 1 {
		t.Fatalf("edits = %#v", result.Files)
	}
	after := string(result.Files[0].After)
	if !strings.Contains(after, `[addons."postgres/jobs"]`) || !strings.Contains(after, `version = "v1.2.0"`) {
		t.Fatalf("TOML =\n%s", after)
	}
	next, err := lazycode.FromFiles("", map[string][]byte{"addons.toml": result.Files[0].After})
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

func TestMultilineArrayRemovalAndTableRemoval(t *testing.T) {
	document, err := tomlcode.Parse([]byte("[entrypoint.app]\nimports = [\n  \"a\",\n  \"b\", # comment\n]\nmodule = \"app\"\n\n[next]\nvalue = true\n"))
	if err != nil {
		t.Fatal(err)
	}
	changed, err := document.Remove("entrypoint.app", "imports")
	if err != nil || !changed {
		t.Fatalf("Remove = %v, %v", changed, err)
	}
	if strings.Contains(string(document.Bytes()), `"a"`) || !strings.Contains(string(document.Bytes()), `module = "app"`) {
		t.Fatalf("TOML after remove =\n%s", document.Bytes())
	}
	changed, err = document.RemoveTable("next")
	if err != nil || !changed || strings.Contains(string(document.Bytes()), "[next]") {
		t.Fatalf("RemoveTable = %v, %v:\n%s", changed, err, document.Bytes())
	}
}

func TestConservativeParserRejectsAmbiguity(t *testing.T) {
	for _, source := range []string{
		"[a]\nx = 1\n[a]\ny = 2\n",
		"[a]\nx = 1\nx = 2\n",
		"[a]\nx = [\n",
		"[a]\nbad key = 1\n",
	} {
		if _, err := tomlcode.Parse([]byte(source)); err == nil {
			t.Fatalf("Parse(%q) error = nil", source)
		}
	}
}

func TestSetStringUsesTOMLCompatibleEscapes(t *testing.T) {
	document, err := tomlcode.Parse([]byte(""))
	if err != nil {
		t.Fatal(err)
	}
	changed, err := document.SetString("addon", "value", "line\ncontrol\x01")
	if err != nil || !changed {
		t.Fatalf("SetString = %v, %v", changed, err)
	}
	if got := string(document.Bytes()); !strings.Contains(got, `value = "line\ncontrol\u0001"`) {
		t.Fatalf("TOML = %q", got)
	}
}

func TestRawAndTypedSetOperationsSupportOwnershipPlanning(t *testing.T) {
	document, err := tomlcode.Parse([]byte("[service]\nname = \"before\" # keep\nvalues = [\"one\"]\n"))
	if err != nil {
		t.Fatal(err)
	}
	if value, found, err := document.Raw("service", "name"); err != nil || !found || value != `"before"` {
		t.Fatalf("raw value = %q, %t, %v", value, found, err)
	}
	workspace, err := lazycode.FromFiles("", map[string][]byte{"lazy.toml": document.Bytes()})
	if err != nil {
		t.Fatal(err)
	}
	result, err := workspace.Plan(
		tomlcode.SetString("lazy.toml", "service", "name", "after"),
		tomlcode.SetStrings("lazy.toml", "service", "values", []string{"one", "two"}),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 1 || !strings.Contains(string(result.Files[0].After), `name = "after" # keep`) || !strings.Contains(string(result.Files[0].After), `values = ["one", "two"]`) {
		t.Fatalf("typed TOML edits = %#v", result.Files)
	}
}
