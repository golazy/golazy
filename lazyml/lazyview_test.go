package lazyml

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func ExampleNewElement() {
	Beautify = false
	content := NewElement("html", NewAttr("lang", "en"),
		NewElement("head",
			NewElement("title", "Mi pagina")),
		NewElement("body",
			NewElement("h1", "This is my page")),
	)

	content.WriteTo(os.Stdout)

	// Output: <!DOCTYPE html><html lang=en><head><title>Mi pagina</title><body><h1>This is my page</h1>
}

// ttr test render tag
func trt(t *testing.T, what io.WriterTo, expectation string) {
	t.Helper()
	buf := &bytes.Buffer{}
	n, err := what.WriteTo(buf)
	if err != nil {
		t.Error(err)
	}
	if n != int64(len(buf.Bytes())) {
		t.Errorf("Size missmatch. Got %d but was %d", n, len(buf.Bytes()))
	}
	if expectation != buf.String() {
		t.Errorf("Expecting %q Got %q", expectation, buf.String())
	}
}

func TestRendererRenderTo(t *testing.T) {
	trt(t, NewElement("html"), "<!DOCTYPE html><html>")
	trt(t, NewElement("html", NewAttr("lang", "en")), `<!DOCTYPE html><html lang=en>`)
	trt(t, NewElement("meta"), `<meta>`)
}
