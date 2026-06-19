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
		for _, part := range strings.Split(value, ",") {
			if strings.EqualFold(strings.TrimSpace(part), header) {
				return true
			}
		}
	}
	return false
}
