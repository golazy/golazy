package tpl

import (
	"bytes"
	"context"
	"testing"

	"github.com/golazy/golazy/memfs"

	"github.com/golazy/golazy/lazyview"
)

func TestEngine_genTemplate(t *testing.T) {
	views := &lazyview.Views{
		FS: memfs.New().Add("test.tpl", []byte("{{.Name}}")),
		Engines: map[string]lazyview.Engine{
			"tpl": &Engine{},
		},
	}

	buf := &bytes.Buffer{}
	vars := map[string]any{
		"Name": "John",
	}
	err := views.RenderTemplate(context.Background(), buf, vars, "test.tpl")
	if err != nil {
		t.Fatal(err)
	}
	if buf.String() != "John" {
		t.Errorf("expected John, got %s", buf.String())
	}

	// Add your assertions here
}
