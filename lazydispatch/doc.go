// Package lazydispatch owns request middleware dispatch for GoLazy.
//
// lazyapp builds the normal dispatcher chain: route-only middleware for
// controller responses, session persistence, application middleware, generated
// metadata endpoints, route dispatch, and public assets. Most applications
// should configure middleware through lazyapp.Config instead of assembling this
// package directly.
//
// Use lazydispatch directly when composing a custom http.Handler stack with the
// same small middleware interface, response buffering, ETag handling, route-only
// middleware, or static fallback behavior.
package lazydispatch
