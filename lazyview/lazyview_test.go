package lazyview_test

import (
	"bytes"
	"context"
	"embed"
	"io/fs"
	"strings"
	"testing"

	"golazy.dev/lazyview"
	"golazy.dev/lazyview/engines/raw"
	"golazy.dev/lazyview/engines/tpl"
)

//go:embed all:test_views
var testViews embed.FS

type testCase struct {
	lazyview.Options
	Output string
}

var RenderCases = []testCase{
	{Options: lazyview.Options{Action: "index"}, Output: "application/index.html.tpl"},
	{Options: lazyview.Options{Partial: "menu"}, Output: "application/_menu.html.tpl"},
	{Options: lazyview.Options{Controller: "posts", Action: "index"}, Output: "posts/index.html.tpl"},
	{Options: lazyview.Options{Controller: "posts", Action: "index", Formats: []string{"json"}}, Output: "posts/index.json.tpl"},
	{Options: lazyview.Options{Controller: "posts", Action: "index", Accept: "application/json;q=0.99, text/html, application/xhtml+xml, application/xml;q=0.9, image/webp, */*;q=0.8"}, Output: "posts/index.json.tpl"},
	{Options: lazyview.Options{Controller: "posts", Action: "index", Variants: []string{"mobile"}}, Output: "posts/index.html+mobile.tpl"},
	{Options: lazyview.Options{Controller: "posts", Action: "index", Variants: []string{"web", ""}}, Output: "posts/index.html.tpl"},
	{Options: lazyview.Options{Namespace: "admin", Action: "index"}, Output: "application/index.html.tpl"},
	{Options: lazyview.Options{Namespace: "admin", Controller: "posts", Action: "index"}, Output: "admin/posts/index.html.tpl"},
	{Options: lazyview.Options{Namespace: "admin", Controller: "authors", Action: "index"}, Output: "plugin/admin/authors/index.html.tpl"},
}

func TestTemplates(t *testing.T) {

	viewFiles, _ := fs.Sub(testViews, "test_views")
	views := &lazyview.Views{
		FS: viewFiles,
		Engines: map[string]lazyview.Engine{
			"txt": &raw.Engine{},
			"tpl": &tpl.Engine{},
		},
		SearchPaths: []string{"", "plugin"},
	}

	buf := &bytes.Buffer{}
	t.Run("Variables", func(t *testing.T) {
		err := views.RenderTemplate(context.Background(), buf, nil, "posts/index.txt")
		if err != nil {
			t.Fatal(err)
		}
		if buf.String() != "posts index" {
			t.Fatalf("unexpected output: %s", buf.String())
		}
	})

	for _, c := range RenderCases {
		t.Run(c.String(), func(t *testing.T) {
			buf.Reset()
			c.Options.Writer = buf
			err := views.Render(c.Options)
			if err != nil {
				t.Fatal(err)
			}
			if buf.String() != c.Output {
				t.Fatalf("expected %s, got %s", c.Output, buf.String())
			}

		})
	}

	// Get a list of views that support the
}

var LayoutCases = []testCase{
	{Options: lazyview.Options{Action: "index"}, Output: "layouts/application.html.tpl application/index.html.tpl"},
	{Options: lazyview.Options{Namespace: "admin", Action: "index"}, Output: "layouts/admin/application.html.tpl application/index.html.tpl"},
	{Options: lazyview.Options{Controller: "posts", Action: "index"}, Output: "layouts/posts.html.tpl posts/index.html.tpl"},
	{Options: lazyview.Options{Controller: "secret", Action: "index"}, Output: "plugin/layouts/secret.html.tpl application/index.html.tpl"},
	{Options: lazyview.Options{Controller: "secret", Action: "index", Layout: "super"}, Output: "layouts/super.html.tpl application/index.html.tpl"},
}

func TestLayout(t *testing.T) {
	viewFiles, _ := fs.Sub(testViews, "test_views")
	views := &lazyview.Views{
		FS: viewFiles,
		Engines: map[string]lazyview.Engine{
			"txt": &raw.Engine{},
			"tpl": &tpl.Engine{},
		},
		SearchPaths: []string{"", "plugin"},
	}
	buf := &bytes.Buffer{}

	for _, c := range LayoutCases {
		t.Run(c.String(), func(t *testing.T) {
			buf.Reset()
			c.Options.Writer = buf
			c.Options.UseLayout = true
			err := views.Render(c.Options)
			if err != nil {
				t.Fatal(err)
			}
			if strings.TrimSpace(buf.String()) != c.Output {
				t.Fatalf("expected %s, got %s", c.Output, buf.String())
			}

		})
	}

}

func (c testCase) String() string {
	out := []string{}
	if c.Namespace != "" {
		out = append(out, "#"+c.Namespace)
	}
	if c.Controller != "" {
		out = append(out, c.Controller)
	}
	if c.Action != "" {
		out = append(out, c.Action)
	}
	if c.Partial != "" {
		out = append(out, "_"+c.Partial)
	}
	if c.Formats != nil {
		out = append(out, "("+strings.Join(c.Formats, "|")+")")
	}
	if c.Variants != nil {
		out = append(out, ".("+strings.Join(c.Variants, "|")+")")
	}
	path := strings.Join(out, "/")
	if c.Accept != "" {
		path += " Accept: " + c.Accept
	}
	return path
}
