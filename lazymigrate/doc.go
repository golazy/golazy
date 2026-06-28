// Package lazymigrate loads, plans, and applies file-based migrations through
// backend-owned execution.
//
// The package is intentionally backend-agnostic. It loads migration files,
// diffs them against the migrations a backend reports as already applied, and
// asks that backend to run concrete up or down steps. The backend owns the
// migration file format, locking, transactions, schema storage, and metadata
// tables.
//
// Applications can load migrations from an fs.FS, combine application and
// package-provided sources in a Catalog, and then run a Migrator with a chosen
// Backend. The fakemigrator subpackage provides an in-memory backend for tests
// and early command wiring.
package lazymigrate
