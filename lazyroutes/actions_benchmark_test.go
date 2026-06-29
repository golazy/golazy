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

type benchmarkRequestController struct {
	lazycontroller.Base
}

type benchmarkPage struct {
	Title    string
	Subtitle string
}

type benchmarkNav struct {
	Current string
}

var benchmarkStaticOK = []byte("OK")

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

func newBenchmarkRequestController(ctx context.Context) (*benchmarkRequestController, error) {
	base, err := lazycontroller.NewBase(ctx, "benchmark")
	if err != nil {
		return nil, err
	}
	return &benchmarkRequestController{Base: base}, nil
}

func (c *benchmarkRequestController) BeforeAction() error {
	return nil
}

func (c *benchmarkRequestController) GenBenchmarkPage(_ *http.Request) benchmarkPage {
	return benchmarkPage{
		Title:    "GoLazy benchmark",
		Subtitle: "request baseline",
	}
}

func (c *benchmarkRequestController) GenBenchmarkNav(_ benchmarkPage) (benchmarkNav, error) {
	return benchmarkNav{Current: "benchmarks"}, nil
}

func (c *benchmarkRequestController) Static(w http.ResponseWriter, _ benchmarkPage, _ benchmarkNav) error {
	_, err := w.Write(benchmarkStaticOK)
	return err
}

func (c *benchmarkRequestController) WithPartials(page benchmarkPage, nav benchmarkNav) error {
	c.Set("title", page.Title)
	c.Set("subtitle", page.Subtitle)
	c.Set("nav", nav.Current)
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

func BenchmarkControllerBeforeGeneratorsWrite(b *testing.B) {
	scope := newBenchmarkScope(b)
	scope.Get("/benchmarks/static", newBenchmarkRequestController, (*benchmarkRequestController).Static)
	request := httptest.NewRequest(http.MethodGet, "/benchmarks/static", nil)
	response := newBenchmarkResponseWriter()
	scope.ServeHTTP(response, request)
	if response.status != 0 || response.bytes != len(benchmarkStaticOK) {
		b.Fatalf("static benchmark response status=%d bytes=%d, want implicit 200 with %d bytes", response.status, response.bytes, len(benchmarkStaticOK))
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		response.Reset()
		scope.ServeHTTP(response, request)
	}
}

func BenchmarkControllerBeforeGeneratorsAutoRenderPartials(b *testing.B) {
	scope := newBenchmarkScope(b)
	scope.Get("/benchmarks/partials", newBenchmarkRequestController, (*benchmarkRequestController).WithPartials)
	request := httptest.NewRequest(http.MethodGet, "/benchmarks/partials", nil)
	response := newBenchmarkResponseWriter()
	scope.ServeHTTP(response, request)
	if response.status != http.StatusOK || response.bytes == 0 {
		b.Fatalf("partials benchmark response status=%d bytes=%d, want 200 with body", response.status, response.bytes)
	}

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
		"layouts/app.html.tpl":            {Data: []byte(`<html><body>{{.content}}</body></html>`)},
		"benchmark/rendered.html.tpl":     {Data: []byte(`{{.message}}`)},
		"benchmark/withpartials.html.tpl": {Data: []byte(`{{partial "header" .}}{{partial "summary" .}}{{partial "footer" .}}`)},
		"benchmark/_header.html.tpl":      {Data: []byte(`<header><h1>{{.title}}</h1></header>`)},
		"benchmark/_summary.html.tpl":     {Data: []byte(`<main><p>{{.subtitle}}</p><p>{{.nav}}</p></main>`)},
		"benchmark/_footer.html.tpl":      {Data: []byte(`<footer>OK</footer>`)},
	})
	if err != nil {
		b.Fatal(err)
	}
	if err := renderer.Cache(); err != nil {
		b.Fatal(err)
	}
	return New(lazycontroller.WithRenderer(context.Background(), renderer))
}

type benchmarkResponseWriter struct {
	header http.Header
	status int
	bytes  int
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
	w.bytes += len(data)
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
	w.bytes = 0
}
