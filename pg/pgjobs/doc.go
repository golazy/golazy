// Package pgjobs implements a PostgreSQL backend for golazy.dev/lazyjobs.
//
// Most GoLazy applications install its add-on and migrations with:
//
//	lazy add postgres/jobs
//
// Applications that wire the backend directly can construct it and register
// its migration source with:
//
//	backend := pgjobs.New(pool)
//	migrations := pgjobs.Migrations()
//
// Add-on integrations that compose a shared lazymigrate Catalog can mount the
// filesystem returned by MigrationFiles instead.
package pgjobs
