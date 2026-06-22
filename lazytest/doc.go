// Package lazytest provides HTTP-level test helpers for GoLazy applications.
//
// Use New with a *lazyapp.App when tests need named route helpers or route
// table assertions. Use FromHandler when testing a plain http.Handler. The
// returned App can issue requests through httptest, assert status codes,
// response bodies, headers, content types, JSON payloads, and route-generated
// paths. Client keeps cookies across requests for session flows.
//
// Typical application tests stay at this level instead of calling controller
// actions directly:
//
//	func TestHomePage(t *testing.T) {
//		app := lazytest.New(t, appinit.App())
//		app.Get("/").OK().Contains("Welcome")
//	}
package lazytest
