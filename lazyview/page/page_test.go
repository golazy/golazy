package page

import (
	"embed"
	"io"
	"testing"

	"golazy.dev/lazyview/component"
	"golazy.dev/lazyview/components/turbo"
	"golazy.dev/lazyview/html"
	"golazy.dev/lazyview/nodes"
	"golazy.dev/lazyview/script"
	"golazy.dev/lazyview/static_files"
	"golazy.dev/lazyview/style"
	"golazy.dev/lazyview/viewtest"
)

//go:embed test_assets/*
var FS embed.FS

func TestPage(t *testing.T) {
	nodes.Beautify = false

	expect := func(p *Page, expectation string) {
		t.Helper()
		diff, err := viewtest.TextDiff(p.String(), expectation)
		if err != nil {
			t.Error(err)
		}
		if diff != nil {
			t.Error(*diff)
		}
	}

	// Empty
	expect(&Page{}, "<!DOCTYPE html><html><head><body>")

	// Lang
	expect(&Page{
		Lang: "en",
	}, "<!DOCTYPE html><html lang=en><head><body>")

	// Charset
	expect(&Page{Charset: "utf-8"},
		"<!DOCTYPE html><html><head><meta charset=utf-8><body>")

	// Viewport
	expect(&Page{Viewport: "width=device-width, initial-scale=1"},
		"<!DOCTYPE html><html><head><meta name=viewport content=\"width=device-width, initial-scale=1\"><body>")

	// Title
	expect(&Page{Title: "My Page"},
		"<!DOCTYPE html><html><head><title>My Page</title><body>")

	// Description
	expect(&Page{Description: "My Page"},
		"<!DOCTYPE html><html><head><meta name=description content=\"My Page\"><body>")

	// Keywords
	expect(&Page{Keywords: "My Page"},
		"<!DOCTYPE html><html><head><meta name=keywords content=\"My Page\"><body>")

	// Content
	expect(&Page{Content: "Hello World"},
		"<!DOCTYPE html><html><head><body>Hello World")

	// Script
	expect(&Page{Scripts: []script.Script{{Src: "script.js"}}},
		"<!DOCTYPE html><html><head><script src=script.js type=module></script><body>")
	expect(&Page{Scripts: []script.Script{{Content: `Log("hola");`}}},
		"<!DOCTYPE html><html><head><script type=module>Log(\"hola\");</script><body>")

	// Style
	expect(&Page{Styles: []style.Style{{Href: "style.css"}}},
		"<!DOCTYPE html><html><head><link href=style.css rel=stylesheet><body>")

	// Head
	expect(&Page{Head: []io.WriterTo{html.Script("hola")}},
		"<!DOCTYPE html><html><head><script>hola</script><body>")

	// ImportMap
	expect(&Page{ImportMap: component.ImportMap{"a": "b"}},
		`<!DOCTYPE html><html><head><script type=importmap>{"imports":{"a":"b"}}</script><body>`)
	p := &Page{}
	p.Use(turbo.Component)

	expect(p,
		`<!DOCTYPE html><html><head><script type=importmap>{"imports":{"@hotwired/turbo":"/@hotwired/turbo/dist/turbo.es2017-esm.js"}}</script><script type=module>import * as Turbo from "@hotwired/turbo"; Turbo.start(); </script><body>`)

	expect(&Page{ImportMap: component.ImportMap{"a": "b"}},
		`<!DOCTYPE html><html><head><script type=importmap>{"imports":{"a":"b"}}</script><body>`)
	p.Files = static_files.NewManager(FS, "test_assets")

	expect(p,
		`<!DOCTYPE html><html><head><script type=importmap>{"imports":{"@hotwired/turbo":"/@hotwired/turbo/dist/turbo.es2017-esm-38b060a751ac.js"}}</script><script type=module>import * as Turbo from "@hotwired/turbo"; Turbo.start(); </script><body>`)

}
