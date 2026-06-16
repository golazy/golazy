package lazyroutes

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"golazy.dev/lazycontroller"
	_ "golazy.dev/lazyview/gotmpl"
)

type benchmarkController struct {
	lazycontroller.Base
}

func newBenchmarkController(ctx context.Context) (*benchmarkController, error) {
	base, err := lazycontroller.NewBase(ctx)
	if err != nil {
		return nil, err
	}
	return &benchmarkController{Base: base}, nil
}

func (c *benchmarkController) Index(w http.ResponseWriter, _ *http.Request) error {
	_, err := fmt.Fprint(w, "ok")
	return err
}

func (c *benchmarkController) Rendered(_ http.ResponseWriter, _ *http.Request) error {
	c.Set("message", "ok")
	return nil
}

func BenchmarkControllerActionWrite(b *testing.B) {
	scope := newBenchmarkScope(b)
	scope.Get("/posts", newBenchmarkController, (*benchmarkController).Index)
	request := httptest.NewRequest(http.MethodGet, "/posts", nil)
	response := newBenchmarkResponseWriter()

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		response.Reset()
		scope.ServeHTTP(response, request)
	}
}

func BenchmarkControllerActionAutoRender(b *testing.B) {
	scope := newBenchmarkScope(b)
	scope.Get("/posts", newBenchmarkController, (*benchmarkController).Rendered)
	request := httptest.NewRequest(http.MethodGet, "/posts", nil)
	response := newBenchmarkResponseWriter()

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		response.Reset()
		scope.ServeHTTP(response, request)
	}
}

func newBenchmarkScope(b *testing.B) *Scope {
	b.Helper()
	renderer, err := lazycontroller.NewRenderer(fstest.MapFS{
		"layouts/app.html.tpl":        {Data: []byte(`{{.content}}`)},
		"benchmark/rendered.html.tpl": {Data: []byte(`{{.message}}`)},
	})
	if err != nil {
		b.Fatal(err)
	}
	return New(lazycontroller.WithRenderer(context.Background(), renderer))
}

type benchmarkResponseWriter struct {
	header http.Header
	status int
}

func newBenchmarkResponseWriter() *benchmarkResponseWriter {
	return &benchmarkResponseWriter{
		header: make(http.Header),
	}
}

func (w *benchmarkResponseWriter) Header() http.Header {
	return w.header
}

func (w *benchmarkResponseWriter) Write(data []byte) (int, error) {
	return len(data), nil
}

func (w *benchmarkResponseWriter) WriteHeader(status int) {
	w.status = status
}

func (w *benchmarkResponseWriter) Reset() {
	for key := range w.header {
		delete(w.header, key)
	}
	w.status = 0
}
