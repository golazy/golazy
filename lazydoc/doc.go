// Package lazydoc extracts, stores, loads, and searches Go package
// documentation as a small JSON-friendly index.
//
// LoadDir reads a module directory, gets its module path from go.mod, walks its
// package directories, and uses the standard go/parser and go/doc packages to
// collect package comments, declarations, constants, variables, functions,
// types, and methods. LoadPackagesFromDir does the same package walk when the
// caller already knows the module path. Both functions ignore test packages,
// vendor, testdata, node_modules, .git, and hidden directories.
//
// The model keeps source metadata beside each package or symbol. Source is the
// source file path relative to the module root plus the original line number.
// It intentionally does not store an absolute path, so generated indexes can be
// embedded, committed, or served from another machine. apps/golazy.dev uses this
// metadata to turn package headings and symbol headings into repository links on
// the public package documentation pages.
//
// The data types in this package are deliberately plain structs with JSON tags.
// A build or documentation command can call LoadDir and marshal the resulting
// Index or Version. A web application can later use LoadJSON or LoadJSONBytes
// to load that index without reparsing source code at request time. Index also
// provides small lookup helpers for version, package, symbol, and search pages.
//
// The public GoLazy site uses lazydoc from apps/golazy.dev/cmd/packagedocs to
// generate data/package_docs/*.json, then loads those files through the
// webcontent service for the /packages routes. Most GoLazy applications do not
// need lazydoc at runtime; use it directly when building a package reference,
// search index, local documentation browser, or another module-level docs
// surface.
package lazydoc
