package lazycontroller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"golazy.dev/lazysession"
	"golazy.dev/lazyview"
	_ "golazy.dev/lazyview/gotmpl"
)

func TestSessionSetSavesCookie(t *testing.T) {
	manager := newControllerSessionManager(t)
	base := newControllerSessionBase(t)
	handler := manager.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := base.BindRequest(w, r, lazyview.Route{Controller: "sessions"}); err != nil {
			t.Fatal(err)
		}
		if err := base.SessionSet("message", "hello"); err != nil {
			t.Fatal(err)
		}
		_, _ = fmt.Fprint(w, "saved")
	}))

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if cookies := response.Result().Cookies(); len(cookies) != 1 {
		t.Fatalf("cookies = %d, want 1", len(cookies))
	}
}

func TestSessionGetDoesNotSaveCookie(t *testing.T) {
	manager := newControllerSessionManager(t)
	base := newControllerSessionBase(t)
	write := manager.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := base.BindRequest(w, r, lazyview.Route{Controller: "sessions"}); err != nil {
			t.Fatal(err)
		}
		if err := base.SessionSet("message", "hello"); err != nil {
			t.Fatal(err)
		}
	}))

	response := httptest.NewRecorder()
	write.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	cookies := response.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("initial cookies = %d, want 1", len(cookies))
	}

	read := manager.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := base.BindRequest(w, r, lazyview.Route{Controller: "sessions"}); err != nil {
			t.Fatal(err)
		}
		value, ok, err := base.SessionGet("message")
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Fatal("session value missing")
		}
		_, _ = fmt.Fprint(w, value)
	}))
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.AddCookie(cookies[0])
	response = httptest.NewRecorder()
	read.ServeHTTP(response, request)

	if got, want := response.Body.String(), "hello"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
	if cookies := response.Result().Cookies(); len(cookies) != 0 {
		t.Fatalf("read-only cookies = %d, want 0", len(cookies))
	}
}

func TestFlashGetConsumesAndSavesCookie(t *testing.T) {
	manager := newControllerSessionManager(t)
	base := newControllerSessionBase(t)
	write := manager.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := base.BindRequest(w, r, lazyview.Route{Controller: "sessions"}); err != nil {
			t.Fatal(err)
		}
		if err := base.FlashSet("notice", "saved", "again"); err != nil {
			t.Fatal(err)
		}
	}))

	response := httptest.NewRecorder()
	write.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	cookies := response.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("initial cookies = %d, want 1", len(cookies))
	}

	read := manager.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := base.BindRequest(w, r, lazyview.Route{Controller: "sessions"}); err != nil {
			t.Fatal(err)
		}
		values, err := base.FlashGet("notice")
		if err != nil {
			t.Fatal(err)
		}
		_, _ = fmt.Fprintf(w, "%v", values)
	}))
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.AddCookie(cookies[0])
	response = httptest.NewRecorder()
	read.ServeHTTP(response, request)

	if got, want := response.Body.String(), "[saved again]"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
	if cookies := response.Result().Cookies(); len(cookies) != 1 {
		t.Fatalf("consume cookies = %d, want 1", len(cookies))
	}
}

func newControllerSessionManager(t *testing.T) *lazysession.Manager {
	t.Helper()
	manager, err := lazysession.NewManager(lazysession.Config{
		Name: "test_session",
		KeyPairs: [][]byte{
			[]byte("0123456789abcdef0123456789abcdef"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return manager
}

func newControllerSessionBase(t *testing.T) Base {
	t.Helper()
	renderer, err := NewRenderer(fstest.MapFS{
		"layouts/app.html.tpl": {Data: []byte(`{{.content}}`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	base, err := NewBase(WithRenderer(t.Context(), renderer))
	if err != nil {
		t.Fatal(err)
	}
	return base
}
