package nodes

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func ExampleNewElement() {
	content := NewElement("html", NewAttr("lang", "en"),
		NewElement("head",
			NewElement("title", "Mi pagina")),
		NewElement("body",
			NewElement("h1", "This is my page")),
	)

	content.WriteTo(os.Stdout)

	// Output: <html lang="en"><head><title>Mi pagina</title></head><body><h1>This is my page</h1></body></html>
}

func trt(t *testing.T, what io.WriterTo, expectation string) {
	buf := &bytes.Buffer{}
	n, err := what.WriteTo(buf)
	if err != nil {
		t.Error(err)
	}
	if n != int64(len(buf.Bytes())) {
		t.Error("Size missmatch. Got", n, "but was", len(buf.Bytes()))
	}
	if expectation != buf.String() {
		t.Error("Expecting", expectation, "got", buf.String())
	}
}

func TestRendererRenderTo(t *testing.T) {
	trt(t, NewElement("html"), "<html/>")
	trt(t, NewElement("html", NewAttr("lang", "en")), `<html lang="en"/>`)
	trt(t, NewElement("meta"), `<meta/>`)
}

func ExampleElement() {

}
