package injector

import (
	"bytes"
	"net/http"
	"testing"
)

type TestResponseWriter struct {
	b bytes.Buffer
}

func (w *TestResponseWriter) WrittenData() string {
	return w.b.String()
}

func (w *TestResponseWriter) Header() http.Header {
	panic("not implemented") // TODO: Implement
}

func (w *TestResponseWriter) Write(data []byte) (int, error) {
	return w.b.Write(data)

}

func (w *TestResponseWriter) WriteHeader(statusCode int) {
	panic("not implemented") // TODO: Implement
}

func TestInjector(t *testing.T) {

	rw := &TestResponseWriter{}

	script := "<script>document.write(\"hola\");</script>"
	injector := &Injector{
		ResponseWriter: rw,
		AppendHeader:   script,
	}

	injector.Write([]byte("<html><head>"))
	injector.Write([]byte("<title>hola</title>"))
	injector.Write([]byte("</"))
	injector.Write([]byte("head"))
	injector.Write([]byte(">"))
	injector.Write([]byte("<body>my page</body></html>"))
	injector.Close()
	if rw.WrittenData() != "<html><head><title>hola</title>"+script+"</head><body>my page</body></html>" {
		t.Fatal("Missing append", rw.WrittenData())
	}
}

func TestInjector_Selfclosing(t *testing.T) {
	rw := &TestResponseWriter{}

	script := "<script>document.write(\"hola\");</script>"
	injector := &Injector{
		ResponseWriter: rw,
		AppendHeader:   script,
	}

	injector.Write([]byte("<html><head/><body/></html>"))
	injector.Close()
	expectation := "<html><head>" + script + "</head><body/></html>"
	if rw.WrittenData() != expectation {
		t.Fatalf("\nGot (%d): %q\nWant(%d): %q", len(rw.WrittenData()), rw.WrittenData(), len(expectation), expectation)
	}
}

func TestInjector_NoMatch(t *testing.T) {

	rw := &TestResponseWriter{}

	script := "<script>document.write(\"hola\");</script>"
	injector := &Injector{
		ResponseWriter: rw,
		AppendHeader:   script,
	}

	injector.Write([]byte("<html>"))
	injector.Write([]byte("<body>my page</body></html>"))
	if rw.WrittenData() != "" {
		t.Fatal("Got", rw.WrittenData())
	}
	injector.Close()
	if rw.WrittenData() != "<html><body>my page</body></html>" {
		t.Fatal("Expecting the full string. Got: ", rw.WrittenData())
	}
}
