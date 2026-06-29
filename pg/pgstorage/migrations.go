package pgstorage

import (
	"embed"

	"golazy.dev/lazymigrate"
)

//go:embed migrations/postgres/*
var migrationFiles embed.FS

// Migrations returns the PostgreSQL migrations required by the lazystorage backend.
func Migrations() lazymigrate.Source {
	return lazymigrate.FS{Files: migrationFiles, Dir: "migrations/postgres"}
}
