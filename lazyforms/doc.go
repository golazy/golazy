// Package lazyforms provides model-aware form helpers for GoLazy views.
//
// lazyapp installs these helpers automatically with the application's router so
// templates can build fields and form paths from route metadata. Applications
// that assemble lazyview directly can install the helpers with:
//
//	views.AddHelpers(lazyforms.Helpers(router))
//
// The package uses lazyschema for field names and ids so form generation and
// controller Decode calls stay aligned.
package lazyforms
