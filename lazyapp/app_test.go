package lazyapp

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"golazy.dev/lazyassets"
	"golazy.dev/lazyroutes"
	"golazy.dev/lazysession"
)

func TestAppAddsDynamicETagToRoutesAndAssetETagToPublicFiles(t *testing.T) {
	app := New(Config{
		Name: "test",
		Drawer: func(router *lazyroutes.Scope) {
			router.HandleFunc(http.MethodGet, "/", func(w http.ResponseWriter, _ *http.Request) error {
				_, _ = fmt.Fprint(w, "home")
				return nil
			})
		},
		Public: func() (fs.FS, error) {
			return fstest.MapFS{
				"styles.css": {Data: []byte("body { color: black; }")},
			}, nil
		},
	})

	response := httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("route status = %d, want %d", response.Code, http.StatusOK)
	}
	if response.Header().Get("ETag") == "" {
		t.Fatal("route ETag is empty")
	}

	response = httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/styles.css", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("public status = %d, want %d", response.Code, http.StatusOK)
	}
	if response.Header().Get("ETag") == "" {
		t.Fatal("public ETag is empty")
	}
	if got := response.Header().Get("Cache-Control"); got != "public, max-age=0, must-revalidate" {
		t.Fatalf("public Cache-Control = %q, want asset logical cache policy", got)
	}
}

func TestAppRegistersGeneratedAssetSources(t *testing.T) {
	app := New(Config{
		Name: "test",
		Assets: []lazyassets.Source{
			lazyassets.SourceFunc(func(registry *lazyassets.Registry) error {
				return registry.Add("/generated.js", []byte("console.log('generated');"))
			}),
		},
	})

	response := httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/generated.js", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("generated status = %d, want %d", response.Code, http.StatusOK)
	}
	if response.Body.String() != "console.log('generated');" {
		t.Fatalf("generated body = %q, want generated JavaScript", response.Body.String())
	}
	if app.Assets == nil || app.Assets.Empty() {
		t.Fatal("app Assets registry is empty")
	}
}

func TestAppInstallsSessionManager(t *testing.T) {
	app := New(Config{
		Name: "test",
		Sessions: lazysession.Config{
			Name: "app_session",
			KeyPairs: [][]byte{
				[]byte("0123456789abcdef0123456789abcdef"),
			},
		},
		Drawer: func(router *lazyroutes.Scope) {
			router.HandleFunc(http.MethodGet, "/", func(w http.ResponseWriter, r *http.Request) error {
				session, err := lazysession.Get(r)
				if err != nil {
					return err
				}
				session.Values["visited"] = true
				_, _ = fmt.Fprint(w, "session")
				return nil
			})
		},
	})
	if app.Sessions == nil {
		t.Fatal("app Sessions manager is nil")
	}

	response := httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %d, want 1", len(cookies))
	}
	if cookies[0].Name != "app_session" {
		t.Fatalf("cookie name = %q, want app_session", cookies[0].Name)
	}
}

func TestAppServesStatic500ForUserRouteErrors(t *testing.T) {
	app := New(Config{
		Name: "test",
		Drawer: func(router *lazyroutes.Scope) {
			router.HandleFunc(http.MethodGet, "/returned", func(w http.ResponseWriter, _ *http.Request) error {
				_, _ = fmt.Fprint(w, "partial")
				return errors.New("broken")
			})
			router.HandleFunc(http.MethodGet, "/panic", func(w http.ResponseWriter, _ *http.Request) error {
				_, _ = fmt.Fprint(w, "partial")
				panic("boom")
			})
		},
		Public: func() (fs.FS, error) {
			return fstest.MapFS{
				"500.html": {Data: []byte("<h1>static 500</h1>")},
			}, nil
		},
	})

	for _, path := range []string{"/returned", "/panic"} {
		response := httptest.NewRecorder()
		app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))

		if response.Code != http.StatusInternalServerError {
			t.Fatalf("%s status = %d, want %d", path, response.Code, http.StatusInternalServerError)
		}
		if got, want := response.Body.String(), "<h1>static 500</h1>"; got != want {
			t.Fatalf("%s body = %q, want %q", path, got, want)
		}
	}
}
