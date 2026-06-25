// Package lazyapp composes the framework packages into a runnable GoLazy
// application.
//
// Most applications use lazyapp.New at the application boundary. It wires the
// application context, route scope, view renderer, helper registry, asset
// registry, session manager, robots and configured sitemap endpoints, optional
// control plane, middleware chain, and public asset fallback into one
// http.Handler.
//
// The lower-level packages remain independently usable. Use lazyroutes directly
// for a route table, lazyview and lazycontroller for custom rendering flows, or
// lazyassets for standalone asset serving. Use lazyapp when those pieces should
// behave like a conventional GoLazy application.
package lazyapp
