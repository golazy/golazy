package lazyapp

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"golazy.dev/lazyassets"
	"golazy.dev/lazycache"
	"golazy.dev/lazyconfig"
	"golazy.dev/lazycontroller"
	"golazy.dev/lazycontrolplane"
	"golazy.dev/lazydeps"
	"golazy.dev/lazyerrors"
	"golazy.dev/lazyjobs"
	"golazy.dev/lazyjobs/inmemoryjobs"
	"golazy.dev/lazyroutes"
	"golazy.dev/lazyseo"
	"golazy.dev/lazysession"
	"golazy.dev/lazytelemetry"
	_ "golazy.dev/lazyview/gotmpl"
)

func testViewFS(t *testing.T, files map[string]string) func() (fs.FS, error) {
	t.Helper()
	configureLazyDevViewsForTest(t, files)
	return func() (fs.FS, error) {
		return testMapFS(files), nil
	}
}

type appTestJob struct {
	lazyjobs.BaseJob
	Value string `json:"value"`
}

func (*appTestJob) Kind() string { return "app.test" }

func (*appTestJob) Work(context.Context) error { return nil }

func TestAppRegistersJobsControlPlane(t *testing.T) {
	app := New(Config{
		Jobs: Jobs(lazyjobs.Config{
			Backend: inmemoryjobs.New(),
			Define: func(runner *lazyjobs.JobRunner) {
				runner.MustRegister(&appTestJob{})
			},
			PollInterval: time.Hour,
		}),
	})
	defer app.Jobs.Stop(context.Background())

	if _, err := app.Jobs.Enqueue(context.Background(), &appTestJob{Value: "hello"}); err != nil {
		t.Fatal(err)
	}

	response := httptest.NewRecorder()
	app.ControlPlane.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/jobs", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("jobs status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	body := response.Body.String()
	if !strings.Contains(body, `"kind":"app.test"`) {
		t.Fatalf("jobs body = %s, want registered job kind", body)
	}
}

func TestAppInitializesJobsWithDependencyContext(t *testing.T) {
	type dbKey struct{}
	const dbValue = "postgres"

	app := New(Config{
		Dependencies: func(deps *lazydeps.Scope) error {
			_, err := lazydeps.Service(deps, "postgres", func(ctx context.Context) (context.Context, string, error, context.CancelFunc) {
				return context.WithValue(ctx, dbKey{}, dbValue), dbValue, nil, nil
			})
			return err
		},
		Jobs: func(ctx context.Context) (lazyjobs.Config, error) {
			if got := ctx.Value(dbKey{}); got != dbValue {
				return lazyjobs.Config{}, fmt.Errorf("db value = %#v, want %q", got, dbValue)
			}
			return lazyjobs.Config{
				Backend: inmemoryjobs.New(),
				Define: func(runner *lazyjobs.JobRunner) {
					runner.MustRegister(&appTestJob{})
				},
				PollInterval: time.Hour,
			}, nil
		},
	})
	defer app.Jobs.Stop(context.Background())

	if app.Jobs == nil {
		t.Fatal("app.Jobs is nil")
	}
	if _, ok := app.Context.Value(dbKey{}).(string); !ok {
		t.Fatal("app context is missing dependency value")
	}
}

func TestAppRegistersTelemetryDependency(t *testing.T) {
	t.Setenv("OTEL_SERVICE_NAME", "sample")

	app := New(Config{Name: "test"})
	telemetry, ok := lazytelemetry.FromContext(app.Context)
	if !ok {
		t.Fatal("telemetry missing from app context")
	}
	if telemetry.Config().ServiceName != "sample" {
		t.Fatalf("telemetry service name = %q, want sample", telemetry.Config().ServiceName)
	}
	graph := app.Dependencies.Graph()
	if !graphHasNode(graph, "telemetry") {
		t.Fatalf("dependencies nodes = %#v, want telemetry", graph.Nodes)
	}
	if !graphHasEdge(graph, lazydeps.Edge{From: "app", To: "telemetry"}) {
		t.Fatalf("dependencies edges = %#v, want app -> telemetry", graph.Edges)
	}
}

func testPublicFS(t *testing.T, files map[string]string) func() (fs.FS, error) {
	t.Helper()
	configureLazyDevPublicForTest(t, files)
	return func() (fs.FS, error) {
		return testMapFS(files), nil
	}
}

func graphHasNode(graph lazydeps.Graph, name string) bool {
	for _, node := range graph.Nodes {
		if node == name {
			return true
		}
	}
	return false
}

func graphHasEdge(graph lazydeps.Graph, edge lazydeps.Edge) bool {
	for _, got := range graph.Edges {
		if got == edge {
			return true
		}
	}
	return false
}

func testMapFS(files map[string]string) fstest.MapFS {
	result := make(fstest.MapFS, len(files))
	for name, content := range files {
		result[name] = &fstest.MapFile{Data: []byte(content)}
	}
	return result
}

func TestAppAddsDynamicETagToRoutesAndAssetETagToPublicFiles(t *testing.T) {
	app := New(Config{
		Name: "test",
		Drawer: func(router *lazyroutes.Scope) {
			router.HandleFunc(http.MethodGet, "/", func(w http.ResponseWriter, _ *http.Request) error {
				_, _ = fmt.Fprint(w, "home")
				return nil
			})
		},
		Public: testPublicFS(t, map[string]string{
			"styles.css": "body { color: black; }",
		}),
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
	if lazyDevTestBuild() {
		if got := response.Header().Get("ETag"); got != "" {
			t.Fatalf("public ETag = %q, want empty in lazydev", got)
		}
		if got := response.Header().Get("Cache-Control"); got != "" {
			t.Fatalf("public Cache-Control = %q, want empty in lazydev", got)
		}
	} else {
		if response.Header().Get("ETag") == "" {
			t.Fatal("public ETag is empty")
		}
		if got := response.Header().Get("Cache-Control"); got != "public, max-age=0, must-revalidate" {
			t.Fatalf("public Cache-Control = %q, want asset logical cache policy", got)
		}
	}
}

func TestAppServesDefaultRobotsAndSitemap(t *testing.T) {
	updated := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	app := New(Config{
		Sitemap: SitemapConfig{
			BaseURL: "https://example.com",
			URLs: []SitemapURL{
				{
					Location:    "/",
					LastUpdated: updated,
					ChangeFreq:  "daily",
					Priority:    1,
				},
			},
		},
	})

	response := httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/robots.txt", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("robots status = %d, want %d", response.Code, http.StatusOK)
	}
	robots := response.Body.String()
	for _, expected := range []string{
		"User-agent: *\n",
		"Allow: /\n",
		"Sitemap: https://example.com/sitemap.xml\n",
	} {
		if !strings.Contains(robots, expected) {
			t.Fatalf("robots.txt does not contain %q:\n%s", expected, robots)
		}
	}

	response = httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("sitemap status = %d, want %d", response.Code, http.StatusOK)
	}
	if got, want := response.Header().Get("Last-Modified"), updated.Format(http.TimeFormat); got != want {
		t.Fatalf("Last-Modified = %q, want %q", got, want)
	}
	sitemap := response.Body.String()
	for _, expected := range []string{
		`<loc>https://example.com/</loc>`,
		`<lastmod>2026-06-20</lastmod>`,
		`<changefreq>daily</changefreq>`,
		`<priority>1</priority>`,
	} {
		if !strings.Contains(sitemap, expected) {
			t.Fatalf("sitemap.xml does not contain %q:\n%s", expected, sitemap)
		}
	}

	request := httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil)
	request.Header.Set("If-Modified-Since", updated.Format(http.TimeFormat))
	response = httptest.NewRecorder()
	app.ServeHTTP(response, request)
	if response.Code != http.StatusNotModified {
		t.Fatalf("conditional sitemap status = %d, want %d", response.Code, http.StatusNotModified)
	}
}

func TestAppDoesNotServeSitemapByDefault(t *testing.T) {
	app := New(Config{})

	response := httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/robots.txt", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("robots status = %d, want %d", response.Code, http.StatusOK)
	}
	robots := response.Body.String()
	if !strings.Contains(robots, "User-agent: *\n") || !strings.Contains(robots, "Allow: /\n") {
		t.Fatalf("unexpected robots.txt:\n%s", robots)
	}
	if strings.Contains(robots, "Sitemap:") {
		t.Fatalf("robots.txt advertises sitemap without sitemap config:\n%s", robots)
	}

	response = httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("sitemap status = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func TestAppPanicsWhenDependencyInitializationFails(t *testing.T) {
	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("New did not panic")
		}
		if got := fmt.Sprint(recovered); !strings.Contains(got, "initialize dependencies: database unavailable") {
			t.Fatalf("panic = %q, want dependency initialization error", got)
		}
	}()

	New(Config{
		Dependencies: func(*lazydeps.Scope) error {
			return errors.New("database unavailable")
		},
	})
}

func TestAppInitializesDependencies(t *testing.T) {
	type contextKey struct{}

	app := New(Config{
		Dependencies: func(deps *lazydeps.Scope) error {
			deps.SetContext(context.WithValue(deps.Context(), contextKey{}, "initialized"))
			return nil
		},
	})

	if got := app.Context.Value(contextKey{}); got != "initialized" {
		t.Fatalf("context value = %v, want initialized", got)
	}
}

func TestAppInitializesDefaultCache(t *testing.T) {
	app := New(Config{})
	if app.Cache == nil {
		t.Fatal("app Cache is nil")
	}
	cache, ok := lazycache.FromContext(app.Context)
	if !ok || cache != app.Cache {
		t.Fatalf("context cache = %#v, %v; want app cache", cache, ok)
	}
	if got := app.Cache.Stats().MaxSizeBytes; got != defaultCacheMaxSizeBytes {
		t.Fatalf("default cache MaxSizeBytes = %d, want %d", got, defaultCacheMaxSizeBytes)
	}
	if err := app.Cache.Set("Ada", "user", 1); err != nil {
		t.Fatal(err)
	}
	value, err := lazycache.Get[string](app.Cache, "user", 1)
	if err != nil {
		t.Fatal(err)
	}
	if value != "Ada" {
		t.Fatalf("cache value = %q, want Ada", value)
	}
}

func TestAppServesConfiguredRobotsAndSitemapAlternates(t *testing.T) {
	app := New(Config{
		Robots: RobotsConfig{
			Rules: []RobotsRule{{
				UserAgent: "ExampleBot",
				Disallow:  []string{"/admin"},
			}},
			Sitemaps: []string{"https://cdn.example.com/site.xml"},
		},
		Sitemap: SitemapConfig{
			BaseURL: "https://example.com",
			Sources: []SitemapSource{
				SitemapSourceFunc(func() ([]SitemapURL, error) {
					return []SitemapURL{{
						Location: "/posts/hello",
						Alternates: []SitemapAlternate{
							{Language: "de", Location: "/de/posts/hello"},
						},
					}}, nil
				}),
			},
		},
	})

	response := httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/robots.txt", nil))
	if got := response.Body.String(); !strings.Contains(got, "User-agent: ExampleBot\nDisallow: /admin\n") ||
		!strings.Contains(got, "Sitemap: https://cdn.example.com/site.xml\n") {
		t.Fatalf("unexpected robots.txt:\n%s", got)
	}

	response = httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil))
	sitemap := response.Body.String()
	for _, expected := range []string{
		`<loc>https://example.com/posts/hello</loc>`,
		`<xhtml:link rel="alternate" hreflang="de" href="https://example.com/de/posts/hello"></xhtml:link>`,
	} {
		if !strings.Contains(sitemap, expected) {
			t.Fatalf("sitemap.xml does not contain %q:\n%s", expected, sitemap)
		}
	}
}

func TestAppCanDisableMetadataFiles(t *testing.T) {
	app := New(Config{
		Robots:  RobotsConfig{Disabled: true},
		Sitemap: SitemapConfig{Disabled: true},
	})

	for _, path := range []string{"/robots.txt", "/sitemap.xml"} {
		response := httptest.NewRecorder()
		app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusNotFound {
			t.Fatalf("%s status = %d, want %d", path, response.Code, http.StatusNotFound)
		}
	}
}

func TestAppRegistersSEOHelpers(t *testing.T) {
	app := New(Config{
		Views: testViewFS(t, map[string]string{
			"pages/show.html.tpl":          `{{seo}}`,
			"layouts/app.html.tpl":         `<html lang="{{seo_lang}}"><head>{{.content}}</head></html>`,
			"layouts/turbo_frame.html.tpl": `{{.content}}`,
		}),
		Drawer: func(router *lazyroutes.Scope) {
			router.Get("/", newSEOTestController, (*seoTestController).Show)
		},
		SEO: func(context.Context) []lazyseo.Option {
			return []lazyseo.Option{
				lazyseo.SiteName("Example"),
				lazyseo.Language("de"),
				lazyseo.TwitterCardType("summary"),
			}
		},
	})

	response := httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	body := response.Body.String()
	for _, expected := range []string{
		`<html lang="de">`,
		`<title>Hello - Example</title>`,
		`<meta name="description" content="Page description">`,
		`<meta name="twitter:card" content="summary">`,
		`"@type":"WebPage"`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("response does not contain %q:\n%s", expected, body)
		}
	}
}

func TestAppInitializesDependenciesBeforeSEO(t *testing.T) {
	type contextKey struct{}

	app := New(Config{
		Views: testViewFS(t, map[string]string{
			"pages/show.html.tpl":          `{{seo}}`,
			"layouts/app.html.tpl":         `<html lang="{{seo_lang}}"><head>{{.content}}</head></html>`,
			"layouts/turbo_frame.html.tpl": `{{.content}}`,
		}),
		Dependencies: func(deps *lazydeps.Scope) error {
			deps.SetContext(context.WithValue(deps.Context(), contextKey{}, "Dependency Site"))
			return nil
		},
		Drawer: func(router *lazyroutes.Scope) {
			router.Get("/", newSEOTestController, (*seoTestController).Show)
		},
		SEO: func(ctx context.Context) []lazyseo.Option {
			siteName, _ := ctx.Value(contextKey{}).(string)
			return []lazyseo.Option{
				lazyseo.SiteName(siteName),
				lazyseo.Language("en"),
			}
		},
	})

	response := httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	body := response.Body.String()
	for _, expected := range []string{
		`<html lang="en">`,
		`<title>Hello - Dependency Site</title>`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("response does not contain %q:\n%s", expected, body)
		}
	}
}

type seoTestController struct {
	lazycontroller.Base
}

func newSEOTestController(ctx context.Context) (*seoTestController, error) {
	base, err := lazycontroller.NewBase(ctx, "pages")
	if err != nil {
		return nil, err
	}
	return &seoTestController{Base: base}, nil
}

func (c *seoTestController) Show() error {
	c.Metadata(testPageMetadata{})
	return nil
}

type testPageMetadata struct{}

func (testPageMetadata) Title() string {
	return "Hello"
}

func (testPageMetadata) Description() string {
	return "Page description"
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

type pathForController struct {
	lazycontroller.Base
}

func newPathForController(ctx context.Context) (*pathForController, error) {
	base, err := lazycontroller.NewBase(ctx)
	if err != nil {
		return nil, err
	}
	return &pathForController{Base: base}, nil
}

func (c *pathForController) Show(w http.ResponseWriter, _ *http.Request) error {
	path, err := c.PathFor("posts", "hello world")
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(w, path)
	return err
}

func TestAppWiresRoutePathHelpersIntoControllers(t *testing.T) {
	app := New(Config{
		Name: "test",
		Views: testViewFS(t, map[string]string{
			"layouts/app.html.tpl": "{{.content}}",
		}),
		Drawer: func(router *lazyroutes.Scope) {
			router.Get("/posts/{post_id}", newPathForController, (*pathForController).Show)
		},
	})

	response := httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/posts/current", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if got, want := response.Body.String(), "/posts/hello%20world"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

type defaultErrorController struct {
	lazycontroller.Base
}

func newDefaultErrorController(ctx context.Context) (*defaultErrorController, error) {
	base, err := lazycontroller.NewBase(ctx)
	if err != nil {
		return nil, err
	}
	return &defaultErrorController{Base: base}, nil
}

func (c *defaultErrorController) Show(_ http.ResponseWriter, _ *http.Request) error {
	return lazycontroller.Error(http.StatusTeapot, errors.New("short and stout"))
}

func (c *defaultErrorController) Traced(_ http.ResponseWriter, _ *http.Request) error {
	return lazycontroller.Error(http.StatusInternalServerError, tracedAppError())
}

func TestAppProvidesFrameworkDefaultErrorViews(t *testing.T) {
	app := New(Config{
		Name: "test",
		Drawer: func(router *lazyroutes.Scope) {
			router.Get("/teapot", newDefaultErrorController, (*defaultErrorController).Show)
		},
	})

	response := httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/teapot", nil))

	if response.Code != http.StatusTeapot {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusTeapot)
	}
	body := response.Body.String()
	expected := []string{"GoLazy", "418 I&#39;m a teapot", "glz-error"}
	if lazyDevTestBuild() {
		expected = append(expected, "short and stout")
	} else {
		expected = append(expected, "The request could not be completed")
	}
	for _, want := range expected {
		if !strings.Contains(body, want) {
			t.Fatalf("body does not contain %q:\n%s", want, body)
		}
	}
	if !lazyDevTestBuild() && strings.Contains(body, "short and stout") {
		t.Fatalf("body exposed production error detail:\n%s", body)
	}
}

func TestAppUsesFrameworkDefaultErrorViewWithAppLayout(t *testing.T) {
	app := New(Config{
		Name:              "test",
		ForceDetailErrors: true,
		Views: testViewFS(t, map[string]string{
			"layouts/app.html.tpl": `sample layout {{.content}}`,
		}),
		Drawer: func(router *lazyroutes.Scope) {
			router.Get("/traced", newDefaultErrorController, (*defaultErrorController).Traced)
		},
	})

	response := httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/traced", nil))

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	body := response.Body.String()
	expected := []string{
		"sample layout",
		"glz-error",
		"Backtrace",
		"tracedAppError",
		"app_test.go",
		"load post",
	}
	for _, want := range expected {
		if !strings.Contains(body, want) {
			t.Fatalf("body does not contain %q:\n%s", want, body)
		}
	}
}

func TestAppViewsOverrideFrameworkDefaultErrorViews(t *testing.T) {
	app := New(Config{
		Name: "test",
		Views: testViewFS(t, map[string]string{
			"layouts/app.html.tpl": `user layout {{.content}}`,
			"app/error.html.tpl":   `user error {{.status}} {{.statusText}} {{if .error}}{{.error}}{{else}}safe{{end}}`,
		}),
		Drawer: func(router *lazyroutes.Scope) {
			router.Get("/teapot", newDefaultErrorController, (*defaultErrorController).Show)
		},
	})

	response := httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/teapot", nil))

	if response.Code != http.StatusTeapot {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusTeapot)
	}
	want := "user layout user error 418 I&#39;m a teapot safe"
	if lazyDevTestBuild() {
		want = "user layout user error 418 I&#39;m a teapot 418 I&#39;m a teapot: short and stout"
	}
	if got := response.Body.String(); got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestAppDetailErrorsExposeErrorToErrorView(t *testing.T) {
	app := New(Config{
		Name:              "test",
		ForceDetailErrors: true,
		Views: testViewFS(t, map[string]string{
			"layouts/app.html.tpl": `user layout {{.content}}`,
			"app/error.html.tpl":   `detail {{.status}} {{.error}}`,
		}),
		Drawer: func(router *lazyroutes.Scope) {
			router.Get("/teapot", newDefaultErrorController, (*defaultErrorController).Show)
		},
	})

	response := httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/teapot", nil))

	if response.Code != http.StatusTeapot {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusTeapot)
	}
	if got, want := response.Body.String(), "user layout detail 418 418 I&#39;m a teapot: short and stout"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

//go:noinline
func tracedAppError() error {
	return lazyerrors.New("load post %q", "hello")
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
	if got, want := app.Sessions.Name(), "release-smoke_session"; got != want {
		t.Fatalf("session name = %q, want %q", got, want)
	}
}

func TestAppDerivesSessionNameFromModulePathBase(t *testing.T) {
	app := New(Config{
		Name: "github.com/golazy/letsvotefast",
		Sessions: lazysession.Config{
			Key: "sample-cookie-01",
		},
	})
	if app.Sessions == nil {
		t.Fatal("app Sessions manager is nil")
	}
	if got, want := app.Sessions.Name(), "letsvotefast_session"; got != want {
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
		Public: testPublicFS(t, map[string]string{
			"500.html": "<h1>static 500</h1>",
		}),
	})

	for _, path := range []string{"/returned", "/panic"} {
		response := httptest.NewRecorder()
		app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))

		if response.Code != http.StatusInternalServerError {
			t.Fatalf("%s status = %d, want %d", path, response.Code, http.StatusInternalServerError)
		}
		if lazyDevTestBuild() {
			if got := response.Body.String(); !strings.Contains(got, "broken") && !strings.Contains(got, "boom") {
				t.Fatalf("%s body = %q, want detail error", path, got)
			}
			continue
		}
		if got, want := response.Body.String(), "<h1>static 500</h1>"; got != want {
			t.Fatalf("%s body = %q, want %q", path, got, want)
		}
	}
}

func TestAppCanForceDetailErrors(t *testing.T) {
	app := New(Config{
		Name:              "test",
		ForceDetailErrors: true,
		Drawer: func(router *lazyroutes.Scope) {
			router.HandleFunc(http.MethodGet, "/returned", func(w http.ResponseWriter, _ *http.Request) error {
				_, _ = fmt.Fprint(w, "partial")
				return errors.New("broken")
			})
		},
		Public: testPublicFS(t, map[string]string{
			"500.html": "<h1>static 500</h1>",
		}),
	})

	response := httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/returned", nil))

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	if got := response.Body.String(); !strings.Contains(got, "broken") {
		t.Fatalf("body = %q, want detail error", got)
	}
}

func TestAppInstallsTelemetryFromOTELEnvironment(t *testing.T) {
	unsetOTELForTest(t)
	t.Setenv("OTEL_SERVICE_NAME", "sample")

	app := New(Config{
		Name: "test",
		Drawer: func(router *lazyroutes.Scope) {
			router.HandleFunc(http.MethodGet, "/", func(w http.ResponseWriter, r *http.Request) error {
				if got := r.Header.Get("X-Request-ID"); got != "" {
					t.Fatalf("request header X-Request-ID = %q, want unset", got)
				}
				_, _ = fmt.Fprint(w, "ok")
				return nil
			})
		},
	})

	response := httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))

	if response.Header().Get("X-Request-ID") == "" {
		t.Fatal("response X-Request-ID is empty")
	}
}

func TestAppSkipsTelemetryWhenOTELSDKDisabled(t *testing.T) {
	unsetOTELForTest(t)
	t.Setenv("OTEL_SDK_DISABLED", "true")
	t.Setenv("OTEL_SERVICE_NAME", "sample")

	app := New(Config{
		Name: "test",
		Drawer: func(router *lazyroutes.Scope) {
			router.HandleFunc(http.MethodGet, "/", func(w http.ResponseWriter, _ *http.Request) error {
				_, _ = fmt.Fprint(w, "ok")
				return nil
			})
		},
	})

	response := httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))

	if got := response.Header().Get("X-Request-ID"); got != "" {
		t.Fatalf("response X-Request-ID = %q, want empty", got)
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
		{name: "unset", want: defaultListenAddr},
		{name: "port only", port: " 9191 ", want: ":9191"},
		{name: "addr overrides port", addr: "127.0.0.1:8181", port: "9191", want: "127.0.0.1:8181"},
		{name: "numeric addr", addr: "8181", want: ":8181"},
		{name: "all interfaces", addr: ":8181", want: ":8181"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			unsetenv(t, "ADDR", "PORT")
			if test.addr != "" {
				t.Setenv("ADDR", test.addr)
			}
			if test.port != "" {
				t.Setenv("PORT", test.port)
			}
			reloadEnvironmentForTest(t)

			if got := listenAddr(); got != test.want {
				t.Fatalf("listenAddr() = %q, want %q", got, test.want)
			}
		})
	}
}

func unsetenv(t *testing.T, names ...string) {
	t.Helper()
	oldValues := make(map[string]string, len(names))
	hadValues := make(map[string]bool, len(names))
	for _, name := range names {
		oldValues[name], hadValues[name] = os.LookupEnv(name)
		if err := os.Unsetenv(name); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		for _, name := range names {
			if hadValues[name] {
				_ = os.Setenv(name, oldValues[name])
			} else {
				_ = os.Unsetenv(name)
			}
		}
	})
}

func unsetOTELForTest(t *testing.T) {
	t.Helper()
	var names []string
	oldValues := make(map[string]string)
	hadValues := make(map[string]bool)
	for _, item := range os.Environ() {
		name, _, ok := strings.Cut(item, "=")
		if ok && strings.HasPrefix(name, "OTEL_") {
			names = append(names, name)
			oldValues[name], hadValues[name] = os.LookupEnv(name)
		}
	}
	for _, name := range names {
		if err := os.Unsetenv(name); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		for _, name := range names {
			if hadValues[name] {
				_ = os.Setenv(name, oldValues[name])
			} else {
				_ = os.Unsetenv(name)
			}
		}
	})
}

func reloadEnvironmentForTest(t *testing.T) {
	t.Helper()
	oldEnvironment := environment
	environment = lazyconfig.MustGetenv[struct {
		Addr             string `default:"127.0.0.1:3000"`
		Port             int    `default:"0"`
		ControlPlaneAddr string
		LazyappMigrate   string `var:"LAZYAPP_MIGRATE"`
	}]()
	t.Cleanup(func() {
		environment = oldEnvironment
	})
}

func TestAppDoesNotInstallControlPlaneByDefault(t *testing.T) {
	app := New(Config{
		Name: "test",
		Drawer: func(router *lazyroutes.Scope) {
			router.HandleFunc(http.MethodGet, "/livez", func(w http.ResponseWriter, _ *http.Request) error {
				_, _ = fmt.Fprint(w, "app livez")
				return nil
			})
		},
	})

	response := httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/livez", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if lazyDevTestBuild() {
		if got := response.Body.String(); got != "live\n" {
			t.Fatalf("body = %q, want lazydev control plane route", got)
		}
		return
	}
	if got := response.Body.String(); got != "app livez" {
		t.Fatalf("body = %q, want app route", got)
	}
}

func TestAppKeepsConfiguredControlPlaneOffDirectHandlerByDefault(t *testing.T) {
	app := New(Config{
		Name:         "test",
		ControlPlane: lazycontrolplane.Config{},
		Drawer: func(router *lazyroutes.Scope) {
			router.HandleFunc(http.MethodGet, "/livez", func(w http.ResponseWriter, _ *http.Request) error {
				_, _ = fmt.Fprint(w, "app livez")
				return nil
			})
		},
	})
	if app.ControlPlane == nil {
		t.Fatal("app ControlPlane is nil")
	}

	response := httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/livez", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if lazyDevTestBuild() {
		if got := response.Body.String(); got != "live\n" {
			t.Fatalf("body = %q, want lazydev control plane route", got)
		}
		return
	}
	if got := response.Body.String(); got != "app livez" {
		t.Fatalf("body = %q, want app route", got)
	}
}

func TestConfiguredControlPlaneMountsWhenListenAddressesMatch(t *testing.T) {
	app := New(Config{
		Name:         "test",
		ControlPlane: lazycontrolplane.Config{},
		Drawer: func(router *lazyroutes.Scope) {
			router.HandleFunc(http.MethodGet, "/livez", func(w http.ResponseWriter, _ *http.Request) error {
				_, _ = fmt.Fprint(w, "app livez")
				return nil
			})
		},
	})

	appHandler, controlHandler := app.handlersForListen(defaultListenAddr, "3000", true)
	if controlHandler != nil {
		t.Fatal("control handler is not nil for matching app and control-plane addresses")
	}

	response := httptest.NewRecorder()
	appHandler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/livez", nil))
	if got := response.Body.String(); got != "live\n" {
		t.Fatalf("body = %q, want control plane route", got)
	}
}

func TestAppMountsPrometheusMetricsFromOTELEnv(t *testing.T) {
	t.Setenv("OTEL_SDK_DISABLED", "false")
	t.Setenv("OTEL_METRICS_EXPORTER", "prometheus")
	t.Setenv("CONTROL_PLANE_ADDR", defaultListenAddr)
	reloadEnvironmentForTest(t)

	app := New(Config{
		Name: "test",
		Drawer: func(router *lazyroutes.Scope) {
			router.HandleFunc(http.MethodGet, "/", func(w http.ResponseWriter, _ *http.Request) error {
				_, _ = fmt.Fprint(w, "ok")
				return nil
			})
		},
	})
	if app.ControlPlane == nil {
		t.Fatal("app ControlPlane is nil")
	}

	response := httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("route status = %d, want %d", response.Code, http.StatusOK)
	}
	if err := app.Cache.Set("Ada", "user"); err != nil {
		t.Fatal(err)
	}
	if _, err := app.Cache.Get("user"); err != nil {
		t.Fatal(err)
	}

	response = httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("metrics status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	body := response.Body.String()
	for _, want := range []string{
		"# TYPE http_server_requests_total counter\n",
		`http_server_requests_total{method="GET",route="/",status_class="2xx"} 1` + "\n",
		"# TYPE http_server_request_duration_seconds histogram\n",
		`http_server_request_duration_seconds_bucket{action="unknown",controller="unknown",le="+Inf",method="GET",route="/",status_class="2xx"} 1` + "\n",
		"golazy_cache_enabled 1\n",
		"golazy_cache_hits_total 1\n",
		"golazy_cache_sets_total 1\n",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("metrics body missing %q in:\n%s", want, body)
		}
	}
}

func TestControlPlaneListenAddr(t *testing.T) {
	t.Setenv("CONTROL_PLANE_ADDR", "9090")
	reloadEnvironmentForTest(t)

	addr, ok := controlPlaneListenAddr()
	if !ok {
		t.Fatal("controlPlaneListenAddr() ok = false, want true")
	}
	if addr != ":9090" {
		t.Fatalf("controlPlaneListenAddr() = %q, want :9090", addr)
	}
}

func TestControlPlaneForListenActivatesFromEnvAddress(t *testing.T) {
	app := New(Config{Name: "test"})
	if !lazyDevTestBuild() && app.controlPlaneForListen(false) != nil {
		t.Fatal("control plane active without config or CONTROL_PLANE_ADDR")
	}
	if lazyDevTestBuild() && app.controlPlaneForListen(false) == nil {
		t.Fatal("lazydev control plane is nil")
	}
	if app.controlPlaneForListen(true) == nil {
		t.Fatal("control plane is nil with CONTROL_PLANE_ADDR set")
	}

	configured := New(Config{
		Name:         "test",
		ControlPlane: lazycontrolplane.Config{},
	})
	if lazyDevTestBuild() && configured.controlPlaneForListen(false) != configured.ControlPlane {
		t.Fatal("configured control plane was not reused")
	}
	if !lazyDevTestBuild() && configured.controlPlaneForListen(false) != nil {
		t.Fatal("configured control plane active without CONTROL_PLANE_ADDR")
	}
}

func TestHandlersForListenMountControlPlaneWhenAddressesMatch(t *testing.T) {
	app := New(Config{Name: "test"})

	appHandler, controlHandler := app.handlersForListen(defaultListenAddr, "3000", true)
	if controlHandler != nil {
		t.Fatal("control handler is not nil for matching app and control-plane addresses")
	}

	response := httptest.NewRecorder()
	appHandler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/livez", nil))
	if got := response.Body.String(); got != "live\n" {
		t.Fatalf("app handler /livez body = %q, want control plane", got)
	}
}

func TestHandlersForListenSeparateControlPlaneWhenAddressesDiffer(t *testing.T) {
	app := New(Config{
		Name:         "test",
		ControlPlane: lazycontrolplane.Config{},
		Drawer: func(router *lazyroutes.Scope) {
			router.HandleFunc(http.MethodGet, "/livez", func(w http.ResponseWriter, _ *http.Request) error {
				_, _ = fmt.Fprint(w, "app livez")
				return nil
			})
		},
	})

	appHandler, controlHandler := app.handlersForListen(defaultListenAddr, ":9090", true)
	if controlHandler == nil {
		t.Fatal("control handler is nil for separate control-plane address")
	}

	response := httptest.NewRecorder()
	appHandler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/livez", nil))
	if got := response.Body.String(); got != "app livez" {
		t.Fatalf("app handler /livez body = %q, want app route", got)
	}

	response = httptest.NewRecorder()
	controlHandler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/livez", nil))
	if got := response.Body.String(); got != "live\n" {
		t.Fatalf("control handler /livez body = %q, want control plane", got)
	}

	response = httptest.NewRecorder()
	controlHandler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("control handler / status = %d, want %d", response.Code, http.StatusOK)
	}
	if got := response.Body.String(); !strings.Contains(got, "GoLazy Control Plane") || !strings.Contains(got, "/livez") {
		t.Fatalf("control handler / body = %q, want control-plane index", got)
	}
}

func TestHandlersForListenMountsPprofOnSeparateControlPlaneAddress(t *testing.T) {
	app := New(Config{
		Name:         "test",
		ControlPlane: lazycontrolplane.Config{},
	})

	appHandler, controlHandler := app.handlersForListen(defaultListenAddr, ":9090", true)
	if controlHandler == nil {
		t.Fatal("control handler is nil for separate control-plane address")
	}
	if !app.ControlPlane.HandlesPath("/debug/pprof/profile") {
		t.Fatal("configured control plane does not handle pprof after separate listen setup")
	}

	response := httptest.NewRecorder()
	controlHandler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("control handler pprof status = %d, want %d", response.Code, http.StatusOK)
	}

	response = httptest.NewRecorder()
	appHandler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil))
	if response.Code == http.StatusOK {
		t.Fatal("app handler served pprof on the public listener")
	}
}

func TestHandlersForListenDoesNotMountPprofWhenControlPlaneAddressMatches(t *testing.T) {
	app := New(Config{
		Name:         "test",
		ControlPlane: lazycontrolplane.Config{},
	})

	appHandler, controlHandler := app.handlersForListen(defaultListenAddr, "3000", true)
	if controlHandler != nil {
		t.Fatal("control handler is not nil for matching app and control-plane addresses")
	}
	if app.ControlPlane.HandlesPath("/debug/pprof/") {
		t.Fatal("control plane handles pprof when mounted on the app listener")
	}

	response := httptest.NewRecorder()
	appHandler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil))
	if response.Code == http.StatusOK {
		t.Fatal("same-listener handler served pprof automatically")
	}
}

func TestHandlersForListenSameAddressLeavesAppRoot(t *testing.T) {
	app := New(Config{
		Name:         "test",
		ControlPlane: lazycontrolplane.Config{},
		Drawer: func(router *lazyroutes.Scope) {
			router.HandleFunc(http.MethodGet, "/", func(w http.ResponseWriter, _ *http.Request) error {
				_, _ = fmt.Fprint(w, "app root")
				return nil
			})
		},
	})

	appHandler, controlHandler := app.handlersForListen(defaultListenAddr, "3000", true)
	if controlHandler != nil {
		t.Fatal("control handler is not nil for matching app and control-plane addresses")
	}

	response := httptest.NewRecorder()
	appHandler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if got := response.Body.String(); got != "app root" {
		t.Fatalf("app handler / body = %q, want app root", got)
	}
}

func TestSameListenAddr(t *testing.T) {
	tests := []struct {
		name  string
		left  string
		right string
		want  bool
	}{
		{name: "same normalized port", left: ":3000", right: "3000", want: true},
		{name: "same default addr", left: defaultListenAddr, right: "3000", want: true},
		{name: "wildcard overlaps localhost", left: "0.0.0.0:3000", right: "127.0.0.1:3000", want: true},
		{name: "localhost aliases overlap", left: "localhost:3000", right: "127.0.0.1:3000", want: true},
		{name: "different ports", left: ":3000", right: ":3001", want: false},
		{name: "different concrete hosts", left: "127.0.0.1:3000", right: "192.0.2.10:3000", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := sameListenAddr(test.left, test.right); got != test.want {
				t.Fatalf("sameListenAddr(%q, %q) = %v, want %v", test.left, test.right, got, test.want)
			}
		})
	}
}
