package gotmpl

import (
	"html/template"
	"testing"

	"golazy.dev/lazyview"
)

func TestTemplateVariablesReusesMapWithoutFragments(t *testing.T) {
	variables := map[string]any{"name": "Ada"}

	got := templateVariables(variables)
	got["name"] = "Grace"

	if variables["name"] != "Grace" {
		t.Fatal("templateVariables copied a map that did not need fragment conversion")
	}
}

func TestTemplateVariablesCopiesWhenFragmentsNeedConversion(t *testing.T) {
	variables := map[string]any{
		"name": "Ada",
		"content": lazyview.Fragment{
			Body:        "<strong>OK</strong>",
			ContentType: "text/html; charset=utf-8",
		},
	}

	got := templateVariables(variables)
	got["name"] = "Grace"

	if variables["name"] != "Ada" {
		t.Fatal("templateVariables reused a map that needed fragment conversion")
	}
	if _, ok := got["content"].(template.HTML); !ok {
		t.Fatalf("converted content = %T, want template.HTML", got["content"])
	}
	if _, ok := variables["content"].(lazyview.Fragment); !ok {
		t.Fatalf("source content = %T, want lazyview.Fragment", variables["content"])
	}
}
