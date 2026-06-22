// Package lazyroutes provides the GoLazy route scope, named route helpers, REST
// resources, request route metadata, and controller action binding.
//
// In a normal app, lazyapp.New creates the root Scope and passes it to the
// application's Draw function. Routes register controller constructors and
// actions; the router constructs controllers from context, binds request-local
// state, runs optional BeforeAction hooks, and renders matching views when an
// action returns without writing a response.
//
// The router can also be used directly as an http.Handler. HandleFunc is useful
// for small route tables or low-level tests, while Resources and Namespace
// provide the conventional GoLazy application shape.
package lazyroutes
