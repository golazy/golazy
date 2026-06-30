// Package lazyworkers registers browser worker scripts for GoLazy
// applications.
//
// A Registry records service workers, dedicated web workers, shared workers,
// and future worker-like scripts in one place. It can serve generated scripts,
// expose template helpers for paths and registration snippets, and publish the
// registered inventory to lazydev tooling.
//
// Conventional GoLazy applications get a Registry through lazyapp.Config.
// Packages such as lazypwa can register their own workers there, while
// applications can also register independent worker scripts directly. The
// package is still usable without lazyapp when a custom net/http stack wants to
// serve or inspect worker scripts.
package lazyworkers
