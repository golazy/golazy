// Package lazytest provides HTTP-level test helpers for GoLazy applications
// and plain net/http handlers.
//
// The package exercises the same handler that serves the application at
// runtime, but it does so through httptest instead of opening a port. That
// makes it a good boundary for behavior that depends on lazyapp composition:
// routing, public files, lazycontroller construction, rendering through
// lazyview, session cookies, response headers, redirects, method errors, and
// asset URLs.
//
// Use New with a *lazyapp.App when the test should see the application's
// lazyroutes.Scope. New stores app.Router on the test wrapper, so PathFor,
// GetPath, and Routes use the same named routes lazycontroller.PathFor and the
// view route helpers use in a running app. Use FromHandler for a plain
// http.Handler, or pass WithRouter when that handler still has a route scope
// that should be available to assertions.
//
// App issues one request at a time and returns a Response with fluent
// assertions for status codes, body text, headers, content type, JSON payloads,
// cookies, and regular-expression matches. Check runs a compact table of
// HTTP-level cases and creates subtests when the supplied testing.TB supports
// Run. Client keeps cookies between requests, which is the normal choice for
// login, flash, lazysession, and other multi-request flows.
//
// Most controller behavior should be tested at this level instead of calling
// action methods directly. Direct controller tests are still useful for narrow
// constructor or dependency checks, but lazytest verifies the wiring between
// lazyapp, lazyroutes, lazycontroller, lazydispatch, lazyview, and public-file
// handling together.
//
// Typical application tests stay at this level instead of calling controller
// actions directly:
//
//	func TestHomePage(t *testing.T) {
//		app := lazytest.New(t, appinit.App())
//
//		app.Get("/").OK().
//			ContentType("text/html").
//			Contains("Welcome")
//	}
//
// Standalone handler tests use the same assertions without requiring a GoLazy
// app:
//
//	func TestJSONHandler(t *testing.T) {
//		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//			w.Header().Set("Content-Type", "application/json")
//			w.Write([]byte(`{"ok":true}`))
//		})
//		app := lazytest.FromHandler(t, handler)
//
//		var payload struct {
//			OK bool `json:"ok"`
//		}
//		app.Get("/", lazytest.Accept("application/json")).
//			OK().
//			ContentType("application/json").
//			JSON(&payload)
//		if !payload.OK {
//			t.Fatal("expected ok payload")
//		}
//	}
package lazytest
