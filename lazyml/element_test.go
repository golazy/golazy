package lazyml

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestElementWriteTo(t *testing.T) {
	Beautify = true

	test := func(e Element, expectation string) {
		t.Helper()
		lines := strings.Split(expectation, "\n")
		if len(lines) > 1 {
			lines = lines[1:]

			l := strings.TrimLeft(lines[0], " \t")
			pad := lines[0][0 : len(lines[0])-len(l)]

			for i, l := range lines {
				lines[i] = strings.TrimPrefix(l, pad)
			}
		}
		expectation = strings.Join(lines, "\n")

		if expectation != e.String() {
			t.Errorf("\nExpecting -----\n%s\nGot --------\n%s\n------\n%q\n%q", expectation, e.String(), expectation, e.String())
		}
	}

	test(NewElement("html", NewAttr("lang", "en"),
		NewElement("head",
			NewElement("title", "Mi pagina")),
		NewElement("body",
			NewElement("h1", "This is my page"),
			NewElement("br")),
	), `
  <!DOCTYPE html>
  <html lang=en>
    <head>
      <title>Mi pagina</title>
  
    <body>
      <h1>This is my page</h1>
      <br>
  
  `)

}

func ExampleBeautify() {
	Beautify = true
	defer (func() {
		Beautify = false
	})()

	NewElement("html", NewAttr("lang", "en"),
		NewElement("head",
			NewElement("title", "Mi pagina")),
		NewElement("body",
			NewElement("h1", "This is my page"),
			NewElement("br")),
	).WriteTo(os.Stdout)

	// Output:
	// <!DOCTYPE html>
	// <html lang=en>
	//   <head>
	//     <title>Mi pagina</title>
	//
	//   <body>
	//     <h1>This is my page</h1>
	//     <br>
	//
	//

}

func testElement(t *testing.T, title, expectation string, e Element) {
	t.Helper()
	t.Run(title, func(t *testing.T) {
		t.Helper()
		b := &bytes.Buffer{}
		e.WriteTo(b)
		if b.String() != expectation {
			t.Errorf("Expected %q Got %q", expectation, b.String())
		}
	})
}

func TestAdd(t *testing.T) {
	Beautify = false
	testElement(t, "have attributes", `<div id=hola></div>`, NewElement("div", NewAttr("id", "hola")))
	testElement(t, "have attributes and content", `<div id=hola>hey</div>`, NewElement("div", NewAttr("id", "hola"), "hey"))
	testElement(t, "escapes string", `<div>&lt;b&gt;hola&lt;/b&gt;</div>`, NewElement("div", "<b>hola</b>"))
	testElement(t, "have raw content", `<div><b>hola</b></div>`, NewElement("div", Raw(`<b>hola</b>`)))
}

func TestElementScript(t *testing.T) {
	Beautify = false
	testElement(t, "script", `<script>console.Log("hola");</script>`, NewElement("script", Raw("console.Log(\"hola\");")))
	testElement(t, "script with type", `<script type=text/javascript>var a = 1;</script>`, NewElement("script", NewAttr("type", "text/javascript"), "var a = 1;"))
}
