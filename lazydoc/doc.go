// Package lazydoc extracts, stores, loads, and searches Go package
// documentation for the GoLazy website and documentation tools.
//
// It reads source directories with the standard go/doc and go/parser packages
// and converts package comments, declarations, methods, values, and examples
// into JSON-friendly models. The website can then load the generated index and
// present package pages without parsing source code at request time.
//
// Applications normally do not need lazydoc. Use it when building documentation
// surfaces for GoLazy packages or for another module that wants the same
// lightweight package index model.
package lazydoc
