package pgjobs

import (
	"embed"
	"io/fs"

	"golazy.dev/lazymigrate"
)

//go:embed migrations/postgres/*
var migrationFiles embed.FS

// Migrations returns the PostgreSQL migrations required by the lazyjobs backend.
func Migrations() lazymigrate.Source {
	return lazymigrate.FS{Files: migrationFiles, Dir: "migrations/postgres"}
}

// MigrationFiles returns the embedded PostgreSQL migrations rooted at their
// migration directory. Add-on integrations mount this filesystem into a
// shared lazymigrate Catalog under the postgres/jobs namespace.
func MigrationFiles() fs.FS {
	files, err := fs.Sub(migrationFiles, "migrations/postgres")
	if err != nil {
		panic("pgjobs: embedded migration directory is missing: " + err.Error())
	}
	return files
}
