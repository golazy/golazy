package lazyapp

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestAppInstallsMethodOverrideForRoutes(t *testing.T) {
	app := New(Config{
		Name: "test",
		Drawer: func(router *lazyroutes.Scope) {
			router.HandleFunc(http.MethodPatch, "/cars/{car_id}", func(w http.ResponseWriter, r *http.Request) error {
				_, _ = fmt.Fprintf(w, "patch %s", r.PathValue("car_id"))
				return nil
			})
		},
	})

	request := httptest.NewRequest(http.MethodPost, "/cars/roadster", strings.NewReader("_method=patch&name=Ada"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	app.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if response.Body.String() != "patch roadster" {
		t.Fatalf("body = %q, want patch roadster", response.Body.String())
	}
}

func TestAppInstallsSessionManager(t *testing.T) {
	app := New(Config{
		Name: "test",
		Sessions: lazysession.Config{
			Key: "sample-cookie-01",
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
	if cookies[0].Name != "test_session" {
		t.Fatalf("cookie name = %q, want test_session", cookies[0].Name)
	}
}

func TestAppKeepsConfiguredSessionName(t *testing.T) {
	app := New(Config{
		Name: "test",
		Sessions: lazysession.Config{
			Name: "custom_session",
			Key:  "sample-cookie-01",
		},
	})
	if app.Sessions == nil {
		t.Fatal("app Sessions manager is nil")
	}
	if got, want := app.Sessions.Name(), "custom_session"; got != want {
		t.Fatalf("session name = %q, want %q", got, want)
	}
}

func TestAppDerivesValidSessionNameFromModulePath(t *testing.T) {
	app := New(Config{
		Name: "example.com/release-smoke",
		Sessions: lazysession.Config{
			Key: "sample-cookie-01",
		},
	})
	if app.Sessions == nil {
		t.Fatal("app Sessions manager is nil")
	}
	if got, want := app.Sessions.Name(), "example.com_release-smoke_session"; got != want {
		t.Fatalf("session name = %q, want %q", got, want)
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

func TestMustSubReturnsSubFS(t *testing.T) {
	fsys := fstest.MapFS{
		"public/styles.css": {Data: []byte("body { color: black; }")},
	}
	public := MustSub(fsys, "public")

	sub, err := public()
	if err != nil {
		t.Fatalf("MustSub func returned error: %v", err)
	}
	content, err := fs.ReadFile(sub, "styles.css")
	if err != nil {
		t.Fatalf("ReadFile(styles.css) error = %v", err)
	}
	if got, want := string(content), "body { color: black; }"; got != want {
		t.Fatalf("styles.css = %q, want %q", got, want)
	}
}

func TestMustSubPanicsForInvalidDir(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("MustSub did not panic")
		}
	}()

	MustSub(fstest.MapFS{}, "../public")
}

func TestListenAddr(t *testing.T) {
	tests := []struct {
		name string
		addr string
		port string
		want string
	}{
		{name: "unset", want: ":3000"},
		{name: "port only", port: "9191", want: ":9191"},
		{name: "port with colon", port: ":9191", want: ":9191"},
		{name: "addr overrides port", addr: "127.0.0.1:8181", port: "9191", want: "127.0.0.1:8181"},
		{name: "numeric addr", addr: "8181", want: ":8181"},
		{name: "all interfaces", addr: ":8181", want: ":8181"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("ADDR", test.addr)
			t.Setenv("PORT", test.port)

			if got := listenAddr(); got != test.want {
				t.Fatalf("listenAddr() = %q, want %q", got, test.want)
			}
		})
	}
}
