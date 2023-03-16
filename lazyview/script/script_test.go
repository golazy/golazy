package script

import (
	"bytes"
	"testing"

	"golazy.dev/lazyview/nodes"
)

func NewExpect(t *testing.T) func(s *Script, expected string) {
	t.Helper()
	return func(s *Script, expected string) {
		t.Helper()
		buf := &bytes.Buffer{}
		s.WriteTo(buf)
		if buf.String() != expected {
			t.Errorf("Expected %s, got %s", expected, buf.String())
		}
	}
}

func TestScript_Content(t *testing.T) {
	nodes.Beautify = false
	expect := NewExpect(t)

	expect(&Script{}, "")
	expect(&Script{Content: `Log("hello");`}, `<script type=module>Log("hello");</script>`)
	expect(&Script{Content: `Log("hello");`, Src: "app.js"}, `<script src=app.js type=module></script><script type=module>Log("hello");</script>`)

	expect(&Script{Src: "http"}, "<script src=http type=module></script>")

	expect(&Script{Src: "http", Type: "text/javascript"}, "<script src=http type=text/javascript></script>")

	expect(&Script{Src: "src", Blocking: "render"}, "<script src=src type=module blocking=render></script>")

	expect(&Script{Src: "src", Async: true}, "<script src=src type=module async></script>")

	expect(&Script{Src: "src", CrossOrigin: Anonymous}, "<script src=src type=module crossorigin=anonymous></script>")
	expect(&Script{Src: "src", CrossOrigin: UseCredentials}, "<script src=src type=module crossorigin=use-credentials></script>")

	expect(&Script{Src: "src", Defer: true}, "<script src=src type=module defer></script>")

	expect(&Script{Src: "src", Integrity: "sha256-foo"}, "<script src=src type=module integrity=sha256-foo></script>")

	expect(&Script{Src: "src", Nonce: "foo"}, "<script src=src type=module nonce=foo></script>")

	expect(&Script{Src: "src", Referrerpolicy: NoReferrer}, "<script src=src type=module referrerpolicy=no-referrer></script>")
	expect(&Script{Src: "src", Referrerpolicy: NoReferrerWhenDowngrade}, "<script src=src type=module referrerpolicy=no-referrer-when-downgrade></script>")
	expect(&Script{Src: "src", Referrerpolicy: Origin}, "<script src=src type=module referrerpolicy=origin></script>")
	expect(&Script{Src: "src", Referrerpolicy: OriginWhenCrossOrigin}, "<script src=src type=module referrerpolicy=origin-when-cross-origin></script>")
	expect(&Script{Src: "src", Referrerpolicy: SameOrigin}, "<script src=src type=module referrerpolicy=same-origin></script>")
	expect(&Script{Src: "src", Referrerpolicy: StrictOrigin}, "<script src=src type=module referrerpolicy=strict-origin></script>")
	expect(&Script{Src: "src", Referrerpolicy: StrictOriginWhenCrossOrigin}, "<script src=src type=module referrerpolicy=strict-origin-when-cross-origin></script>")
	expect(&Script{Src: "src", Referrerpolicy: UnsafeURL}, "<script src=src type=module referrerpolicy=unsafe-url></script>")

	expect(&Script{Src: "src", NoModule: true}, "<script src=src nomodule></script>")
}
