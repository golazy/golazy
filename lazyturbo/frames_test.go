package lazyturbo_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"golazy.dev/lazyturbo"
	"golazy.dev/lazyview"
	_ "golazy.dev/lazyview/gotmpl"
)

func TestFrameHelperRendersFramePartial(t *testing.T) {
	views, err := lazyview.New(fstest.MapFS{
		"layouts/app.html.tpl":      {Data: []byte(`<main>{{.content}}</main>`)},
		"posts/index.html.tpl":      {Data: []byte(`{{ turbo_frame "car" . (turbo_src "/cars/1") (turbo_loading "lazy") (turbo_target "_top") (turbo_action "advance") (turbo_refresh_morph) (turbo_autoscroll) (turbo_autoscroll_block "center") (turbo_autoscroll_behavior "smooth") }}`)},
		"posts/_car_frame.html.tpl": {Data: []byte(`<p>{{.name}}</p>`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	views.AddHelpers(lazyturbo.Helpers())
	if err := views.Cache(); err != nil {
		t.Fatal(err)
	}

	var out strings.Builder
	err = views.Render(lazyview.Options{
		Writer:     &out,
		Variables:  map[string]any{"name": "Ada"},
		Controller: "posts",
		Action:     "index",
		UseLayout:  true,
	})
	if err != nil {
		t.Fatal(err)
	}

	want := `<main><turbo-frame id="car" src="/cars/1" loading="lazy" target="_top" data-turbo-action="advance" refresh="morph" autoscroll data-autoscroll-block="center" data-autoscroll-behavior="smooth"><p>Ada</p></turbo-frame></main>`
	if got := out.String(); got != want {
		t.Fatalf("rendered body = %q, want %q", got, want)
	}
}

func TestFrameTagEscapesAttributes(t *testing.T) {
	frame, err := lazyturbo.FrameTag("car", "<p>trusted</p>", lazyturbo.Src(`/cars?name=<Ada>`))
	if err != nil {
		t.Fatal(err)
	}
	want := `<turbo-frame id="car" src="/cars?name=&lt;Ada&gt;"><p>trusted</p></turbo-frame>`
	if frame.Body != want {
		t.Fatalf("frame body = %q, want %q", frame.Body, want)
	}
}

func TestFrameOptionsValidateEnumeratedValues(t *testing.T) {
	_, err := lazyturbo.FrameTag("car", "", lazyturbo.Loading("sometimes"))
	if err == nil {
		t.Fatal("expected invalid loading error")
	}
}

func TestFrameIDRejectsPathCharacters(t *testing.T) {
	if err := lazyturbo.ValidateFrameID("../car"); err == nil {
		t.Fatal("expected invalid frame id error")
	}
}

func TestRequestHelpers(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Turbo-Frame", "car")
	request.Header.Set("Accept", "text/html, text/vnd.turbo-stream.html")
	request.Header.Set("X-Sec-Purpose", "prefetch")

	if got := lazyturbo.FrameID(request); got != "car" {
		t.Fatalf("FrameID = %q, want car", got)
	}
	if !lazyturbo.IsFrameRequest(request) {
		t.Fatal("IsFrameRequest = false, want true")
	}
	if !lazyturbo.AcceptsStream(request) {
		t.Fatal("AcceptsStream = false, want true")
	}
	if !lazyturbo.IsPrefetch(request) {
		t.Fatal("IsPrefetch = false, want true")
	}
}
