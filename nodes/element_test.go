package nodes

import (
	"bytes"
	"os"
	"testing"
)

func ExampleBeautify() {
	Beautify = true
	defer (func() {
		Beautify = false
	})()

	NewElement("html", NewAttr("lang", "en"), NewAttr("base", "/"),
		NewElement("head",
			NewElement("title", "Mi pagina")),
		NewElement("body",
			NewElement("h1", "This is my page"),
			NewElement("br")),
	).WriteTo(os.Stdout)

	// Output:
	// <html lang="en" base="/">
	// <head>
	// <title>Mi pagina</title>
	// </head>
	// <body>
	// <h1>This is my page</h1>
	// <br/>
	// </body>
	// </html>
}

func testElement(t *testing.T, title, expectation string, e Element) {
	t.Run(title, func(t *testing.T) {
		b := &bytes.Buffer{}
		e.WriteTo(b)
		if b.String() != expectation {
			t.Error("got:", b.String())
		}
	})
}

func TestAdd(t *testing.T) {
	testElement(t, "have attributes", `<div id="hola"/>`, NewElement("div", NewAttr("id", "hola")))
	testElement(t, "have attributes and content", `<div id="hola">hey</div>`, NewElement("div", NewAttr("id", "hola"), "hey"))
	testElement(t, "escapes string", `<div>&lt;b&gt;hola&lt;/b&gt;</div>`, NewElement("div", "<b>hola</b>"))
	testElement(t, "have raw content", `<div><b>hola</b></div>`, NewElement("div", Raw(`<b>hola</b>`)))
}
