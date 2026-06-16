package lazysession

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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

func TestNewManagerRejectsMissingStore(t *testing.T) {
	_, err := NewManager(Config{})
	if err == nil || !strings.Contains(err.Error(), "store or key pairs are required") {
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
