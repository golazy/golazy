// Package lazyseo renders common document metadata for GoLazy views.
//
// lazyapp installs the standard SEO helpers when a renderer is configured, and
// lazycontroller.Base exposes convenience methods for setting page metadata
// from controller actions. Use those integrations in normal applications.
//
// The package can also be used directly with any value that supports
// Set(string, any), which keeps metadata rendering independent from controller
// internals and from a specific template engine.
package lazyseo
