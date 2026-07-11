package lazycontroller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"golazy.dev/lazyturbo"
	"golazy.dev/lazyview"
	_ "golazy.dev/lazyview/gotmpl"
)

func TestFormatFromRequest(t *testing.T) {
	tests := []struct {
		name   string
		header http.Header
		want   Format
	}{
		{name: "defaults to html", want: HTML},
		{
			name:   "json accept",
			header: http.Header{"Accept": {"application/json"}},
			want:   JSON,
		},
		{
			name:   "turbo stream accept",
			header: http.Header{"Accept": {lazyturbo.StreamMIME + ", text/html"}},
			want:   TurboStream,
		},
		{
			name: "turbo frame header wins",
			header: http.Header{
				"Accept":      {"application/json"},
				"Turbo-Frame": {"car"},
			},
			want: TurboFrame,
		},
		{
			name:   "png accept",
			header: http.Header{"Accept": {"image/png"}},
			want:   PNG,
		},
		{
			name:   "jpeg accept",
			header: http.Header{"Accept": {"image/jpeg"}},
			want:   JPEG,
		},
		{
			name:   "gif accept",
			header: http.Header{"Accept": {"image/gif"}},
			want:   GIF,
		},
		{
			name:   "generic image accept",
			header: http.Header{"Accept": {"image/webp"}},
			want:   Image,
		},
		{
			name:   "sse accept",
			header: http.Header{"Accept": {"text/event-stream"}},
			want:   SSE,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			request.Header = tt.header.Clone()
			if got := FormatFromRequest(request); got != tt.want {
				t.Fatalf("FormatFromRequest = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderUsesJSONFormatWithoutLayout(t *testing.T) {
	base, response := newRenderTestBase(t, fstest.MapFS{
		"layouts/app.html.tpl": {Data: []byte(`layout {{.content}}`)},
		"posts/show.json.tpl":  {Data: []byte(`{"title":"{{.title}}"}`)},
	})
	request := httptest.NewRequest(http.MethodGet, "/posts/1", nil)
	request.Header.Set("Accept", "application/json")
	if err := base.BindRequest(response, request, lazyview.Route{Controller: "posts", Action: "Show"}); err != nil {
		t.Fatal(err)
	}
	base.Set("title", "Hello")

	if err := base.Render("show"); err != nil {
		t.Fatal(err)
	}

	if got, want := response.Body.String(), `{"title":"Hello"}`; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
	if got, want := response.Header().Get("Content-Type"), "application/json; charset=utf-8"; got != want {
		t.Fatalf("Content-Type = %q, want %q", got, want)
	}
}

func TestRenderUsesTurboStreamForTurboStreamAccept(t *testing.T) {
	base, response := newRenderTestBase(t, fstest.MapFS{
		"layouts/app.html.tpl":        {Data: []byte(`layout {{.content}}`)},
		"posts/show.html.tpl":         {Data: []byte(`<p>{{.title}}</p>`)},
		"posts/show.turbo_stream.tpl": {Data: []byte(`<turbo-stream action="replace" target="post"></turbo-stream>`)},
	})
	request := httptest.NewRequest(http.MethodGet, "/posts/1", nil)
	request.Header.Set("Accept", lazyturbo.StreamMIME+", text/html, application/xhtml+xml")
	if err := base.BindRequest(response, request, lazyview.Route{Controller: "posts", Action: "Show"}); err != nil {
		t.Fatal(err)
	}
	base.Set("title", "Hello")

	if err := base.Render("show"); err != nil {
		t.Fatal(err)
	}

	if got, want := response.Body.String(), `<turbo-stream action="replace" target="post"></turbo-stream>`; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
	if got := response.Header().Get("Content-Type"); !strings.Contains(got, lazyturbo.StreamMIME) {
		t.Fatalf("Content-Type = %q, want Turbo Stream", got)
	}
}

func TestRenderUsesTurboFrameHeader(t *testing.T) {
	base, response := newRenderTestBase(t, fstest.MapFS{
		"layouts/app.html.tpl":      {Data: []byte(`layout {{.content}}`)},
		"posts/show.html.tpl":       {Data: []byte(`<h1>{{.name}}</h1>`)},
		"posts/_car_frame.html.tpl": {Data: []byte(`<p>{{.name}}</p>`)},
	})
	request := httptest.NewRequest(http.MethodGet, "/posts/1", nil)
	request.Header.Set("Turbo-Frame", "car")
	if err := base.BindRequest(response, request, lazyview.Route{Controller: "posts", Action: "Show"}); err != nil {
		t.Fatal(err)
	}
	base.Set("name", "Ada")
	base.SetTurboFrameOptions(lazyturbo.Src("/cars/1"))

	if err := base.Render("show"); err != nil {
		t.Fatal(err)
	}

	want := `<turbo-frame id="car" src="/cars/1"><p>Ada</p></turbo-frame>`
	if got := response.Body.String(); got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
	if got, want := response.Header().Get("Content-Type"), "text/html; charset=utf-8"; got != want {
		t.Fatalf("Content-Type = %q, want %q", got, want)
	}
	for _, header := range []string{"Accept", "Turbo-Frame"} {
		if !headerValuesContain(response.Header().Values("Vary"), header) {
			t.Fatalf("Vary = %#v, want %s", response.Header().Values("Vary"), header)
		}
	}
}

func TestWantsUsesSelectedFormat(t *testing.T) {
	base, response := newRenderTestBase(t, fstest.MapFS{
		"layouts/app.html.tpl": {Data: []byte(`{{.content}}`)},
	})
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Accept", "application/json")
	if err := base.BindRequest(response, request, lazyview.Route{Controller: "posts", Action: "Show"}); err != nil {
		t.Fatal(err)
	}

	err := base.Wants(Formats{
		HTML: func() error {
			response.Write([]byte("html"))
			return nil
		},
		JSON: func() error {
			response.Write([]byte("json"))
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := response.Body.String(), "json"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestRespondUsesSelectedFormat(t *testing.T) {
	base, response := newRenderTestBase(t, fstest.MapFS{
		"layouts/app.html.tpl": {Data: []byte(`{{.content}}`)},
	})
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Accept", "application/json")
	if err := base.BindRequest(response, request, lazyview.Route{Controller: "posts", Action: "Show"}); err != nil {
		t.Fatal(err)
	}

	err := base.Respond(Responses{
		HTML: func() error {
			response.Write([]byte("html"))
			return nil
		},
		JSON: func() error {
			response.Write([]byte("json"))
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := response.Body.String(), "json"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestWantsFallsBackFromTurboFrameToHTML(t *testing.T) {
	base, response := newRenderTestBase(t, fstest.MapFS{
		"layouts/app.html.tpl": {Data: []byte(`{{.content}}`)},
	})
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Turbo-Frame", "car")
	if err := base.BindRequest(response, request, lazyview.Route{Controller: "posts", Action: "Show"}); err != nil {
		t.Fatal(err)
	}

	err := base.Wants(Formats{
		HTML: func() error {
			response.Write([]byte("html"))
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := response.Body.String(), "html"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestWantsUsesLaterAcceptedFormatWhenPreferredIsUnavailable(t *testing.T) {
	base, response := newRenderTestBase(t, fstest.MapFS{
		"layouts/app.html.tpl": {Data: []byte(`{{.content}}`)},
	})
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Accept", "application/json, text/html")
	if err := base.BindRequest(response, request, lazyview.Route{Controller: "posts", Action: "Show"}); err != nil {
		t.Fatal(err)
	}

	err := base.Wants(Formats{
		HTML: func() error {
			response.Write([]byte("html"))
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := response.Body.String(), "html"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestWantsUsesWildcardAccept(t *testing.T) {
	base, response := newRenderTestBase(t, fstest.MapFS{
		"layouts/app.html.tpl": {Data: []byte(`{{.content}}`)},
	})
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Accept", "*/*")
	if err := base.BindRequest(response, request, lazyview.Route{Controller: "posts", Action: "Show"}); err != nil {
		t.Fatal(err)
	}

	err := base.Wants(Formats{
		JSON: func() error {
			response.Write([]byte("json"))
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := response.Body.String(), "json"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestWantsFallsBackFromConcreteImageToImage(t *testing.T) {
	base, response := newRenderTestBase(t, fstest.MapFS{
		"layouts/app.html.tpl": {Data: []byte(`{{.content}}`)},
	})
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Accept", "image/png")
	if err := base.BindRequest(response, request, lazyview.Route{Controller: "posts", Action: "Show"}); err != nil {
		t.Fatal(err)
	}

	err := base.Wants(Formats{
		Image: func() error {
			response.Write([]byte("image"))
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := response.Body.String(), "image"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestWantsUsesConcreteImageForImageWildcard(t *testing.T) {
	base, response := newRenderTestBase(t, fstest.MapFS{
		"layouts/app.html.tpl": {Data: []byte(`{{.content}}`)},
	})
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Accept", "image/*")
	if err := base.BindRequest(response, request, lazyview.Route{Controller: "posts", Action: "Show"}); err != nil {
		t.Fatal(err)
	}

	err := base.Wants(Formats{
		PNG: func() error {
			response.Write([]byte("png"))
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := response.Body.String(), "png"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestWantsReturnsNotAcceptableForUnavailableFormat(t *testing.T) {
	base, response := newRenderTestBase(t, fstest.MapFS{
		"layouts/app.html.tpl": {Data: []byte(`{{.content}}`)},
	})
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Accept", "application/json")
	if err := base.BindRequest(response, request, lazyview.Route{Controller: "posts", Action: "Show"}); err != nil {
		t.Fatal(err)
	}

	err := base.Wants(Formats{
		HTML: func() error { return nil },
	})
	if err == nil {
		t.Fatal("Wants returned nil, want error")
	}
	if got, want := StatusCode(err), http.StatusNotAcceptable; got != want {
		t.Fatalf("StatusCode = %d, want %d", got, want)
	}
}

func TestWantsSelectedFormatControlsRender(t *testing.T) {
	base, response := newRenderTestBase(t, fstest.MapFS{
		"layouts/app.html.tpl":        {Data: []byte(`layout {{.content}}`)},
		"posts/show.html.tpl":         {Data: []byte(`<p>{{.title}}</p>`)},
		"posts/show.turbo_stream.tpl": {Data: []byte(`<turbo-stream action="replace" target="post"></turbo-stream>`)},
	})
	request := httptest.NewRequest(http.MethodGet, "/posts/1", nil)
	request.Header.Set("Accept", lazyturbo.StreamMIME+", text/html, application/xhtml+xml")
	if err := base.BindRequest(response, request, lazyview.Route{Controller: "posts", Action: "Show"}); err != nil {
		t.Fatal(err)
	}
	base.Set("title", "Hello")

	err := base.Wants(Formats{
		HTML: func() error {
			return base.Render("show")
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if got, want := response.Body.String(), `layout <p>Hello</p>`; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
	if got := response.Header().Get("Content-Type"); !strings.Contains(got, "text/html") {
		t.Fatalf("Content-Type = %q, want HTML", got)
	}
	if got := base.Format(); got != TurboStream {
		t.Fatalf("Format after Wants = %q, want request format %q", got, TurboStream)
	}
}

func TestNewFormatRegistersContentTypeAndSuffix(t *testing.T) {
	format := NewFormat(
		"application/x-golazy-format-test",
		As("golazy_format_test"),
		Suffix("glft"),
	)

	got, ok := FormatFromContentType("application/x-golazy-format-test; charset=utf-8")
	if !ok {
		t.Fatal("FormatFromContentType did not resolve custom format")
	}
	if got != format {
		t.Fatalf("FormatFromContentType = %q, want %q", got, format)
	}

	got, ok = FormatFromSuffix(".glft")
	if !ok {
		t.Fatal("FormatFromSuffix did not resolve custom format")
	}
	if got != format {
		t.Fatalf("FormatFromSuffix = %q, want %q", got, format)
	}
}

func TestNewFormatDerivesNameAndSuffix(t *testing.T) {
	format := NewFormat("application/x-golazy-derived-test")
	if got, want := format, Format("x_golazy_derived_test"); got != want {
		t.Fatalf("NewFormat = %q, want %q", got, want)
	}

	got, ok := FormatFromSuffix("x_golazy_derived_test")
	if !ok {
		t.Fatal("FormatFromSuffix did not resolve derived suffix")
	}
	if got != format {
		t.Fatalf("FormatFromSuffix = %q, want %q", got, format)
	}
}

func newRenderTestBase(t *testing.T, files fstest.MapFS) (Base, *httptest.ResponseRecorder) {
	t.Helper()
	renderer, err := NewRenderer(files)
	if err != nil {
		t.Fatal(err)
	}
	ctx := WithRenderer(context.Background(), renderer)
	base, err := NewBase(ctx)
	if err != nil {
		t.Fatal(err)
	}
	return base, httptest.NewRecorder()
}

func headerValuesContain(values []string, header string) bool {
	for _, value := range values {
		for part := range strings.SplitSeq(value, ",") {
			if strings.EqualFold(strings.TrimSpace(part), header) {
				return true
			}
		}
	}
	return false
}
