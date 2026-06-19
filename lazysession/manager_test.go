package lazysession

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golazy.dev/lazydispatch"
	"golazy.dev/lazysse"
)

func TestManagerMiddlewareSavesDefaultSession(t *testing.T) {
	manager, err := NewManager(Config{
		Name: "test_session",
		KeyPairs: [][]byte{
			[]byte("0123456789abcdef0123456789abcdef"),
		},
		Options: &Options{
			Path:     "/",
			MaxAge:   3600,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	handler := manager.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := Get(r)
		if err != nil {
			t.Fatal(err)
		}
		session.Values["message"] = "hello"
		_, _ = fmt.Fprint(w, "saved")
	}))

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %d, want 1", len(cookies))
	}
	if cookies[0].Name != "test_session" {
		t.Fatalf("cookie name = %q, want test_session", cookies[0].Name)
	}
	if !cookies[0].HttpOnly {
		t.Fatal("session cookie is not HttpOnly")
	}

	read := manager.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := Get(r)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = fmt.Fprint(w, session.Values["message"])
	}))
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.AddCookie(cookies[0])
	response = httptest.NewRecorder()
	read.ServeHTTP(response, request)
	if response.Body.String() != "hello" {
		t.Fatalf("body = %q, want hello", response.Body.String())
	}
}

func TestManagerMiddlewareSavesBeforeStreaming(t *testing.T) {
	manager, err := NewManager(Config{
		Name: "test_session",
		KeyPairs: [][]byte{
			[]byte("0123456789abcdef0123456789abcdef"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	handler := lazydispatch.ResponseBuffer().Handler(manager.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := Get(r)
		if err != nil {
			t.Fatal(err)
		}
		session.Values["stream"] = "saved"
		stream, err := lazysse.Start(w, r)
		if err != nil {
			t.Fatal(err)
		}
		defer stream.Close()
		if err := stream.Send(lazysse.Event{Data: []string{"ok"}}); err != nil {
			t.Fatal(err)
		}
	})))

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if got, want := response.Body.String(), "data: ok\n\n"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
	if cookies := response.Result().Cookies(); len(cookies) != 1 {
		t.Fatalf("cookies = %d, want 1", len(cookies))
	}
}

func TestManagerAcceptsCustomStore(t *testing.T) {
	store := &recordingStore{}
	manager, err := NewManager(Config{
		Name:  "custom_session",
		Store: store,
	})
	if err != nil {
		t.Fatal(err)
	}
	if manager.Store() != store {
		t.Fatal("manager did not keep custom store")
	}

	handler := manager.Handler(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		session, err := Get(r)
		if err != nil {
			t.Fatal(err)
		}
		session.Values["set"] = true
	}))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	if store.newCount != 1 {
		t.Fatalf("New calls = %d, want 1", store.newCount)
	}
	if store.saveCount != 1 {
		t.Fatalf("Save calls = %d, want 1", store.saveCount)
	}
}

func TestNewManagerAcceptsSingleKey(t *testing.T) {
	manager, err := NewManager(Config{
		Key: "sample-cookie-01",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := manager.Name(), defaultSessionName; got != want {
		t.Fatalf("manager name = %q, want %q", got, want)
	}
	if _, ok := manager.Store().(*CookieStore); !ok {
		t.Fatalf("store = %T, want *CookieStore", manager.Store())
	}
}

func TestDeriveKeyExpandsShortKeys(t *testing.T) {
	got := deriveKey("sample-cookie-01")
	want := sha256.Sum256([]byte("sample-cookie-01"))
	if !bytes.Equal(got, want[:]) {
		t.Fatal("derived key does not match SHA-256")
	}
	if len(got) != sha256.Size {
		t.Fatalf("derived key length = %d, want %d", len(got), sha256.Size)
	}
}

func TestNewManagerRejectsMissingStore(t *testing.T) {
	_, err := NewManager(Config{})
	if err == nil || !strings.Contains(err.Error(), "store, key, or key pairs are required") {
		t.Fatalf("error = %v", err)
	}
}

func TestNewManagerRejectsInvalidName(t *testing.T) {
	_, err := NewManager(Config{
		Name: "bad:name",
		KeyPairs: [][]byte{
			[]byte("0123456789abcdef0123456789abcdef"),
		},
	})
	if err == nil || !strings.Contains(err.Error(), "invalid session name") {
		t.Fatalf("error = %v", err)
	}
}

type recordingStore struct {
	newCount  int
	saveCount int
}

func (s *recordingStore) Get(r *http.Request, name string) (*Session, error) {
	return GetRegistry(r).Get(s, name)
}

func (s *recordingStore) New(_ *http.Request, name string) (*Session, error) {
	s.newCount++
	return NewSession(s, name), nil
}

func (s *recordingStore) Save(_ *http.Request, _ http.ResponseWriter, _ *Session) error {
	s.saveCount++
	return nil
}
