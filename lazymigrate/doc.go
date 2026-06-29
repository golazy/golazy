// Package lazymigrate loads, plans, and applies ordered file-based migrations
// through backend-owned execution.
//
// The package is intentionally backend-agnostic. It knows how to collect source
// migrations, compare them with the migration IDs a backend reports as already
// applied, and build an up, down, or redo plan. The backend owns the concrete
// migration language, the meaning of up and down sections, locking,
// transactions, metadata tables, and schema load or dump support.
//
// A Source returns migration files. FS reads direct children from a directory
// in an fs.FS, skips nested directories and migrations.toml, rejects Go
// migration files, and requires each filename to contain a sortable timestamp.
// ForDatabase is the common layout helper for migrations/<database>, for
// example migrations/postgres. File extensions are ignored for migration IDs:
// 202606280001_create_documents.sql becomes ID
// 202606280001_create_documents. Prefix and Timestamp are parsed only for
// stable ordering; duplicate IDs across all loaded sources are rejected.
//
// Catalog is useful when an application combines its own migrations with
// package-provided migrations. PostgreSQL service packages in golazy.dev/pg,
// such as pgjobs, pgfiles, pgmedia, and pgstorage, expose embedded migrations
// as lazymigrate.Source values so an application can add them to the same
// catalog as application migrations.
//
// A Migrator is the planner and executor. List reports applied source
// migrations, pending source migrations, and missing backend migrations.
// PlanUp, PlanDown, and PlanRedo return the Step values that would run without
// touching the backend. Up, Down, and Redo build the plan and then call Apply.
// Apply calls Backend.Setup once before running the plan and passes each Step
// to Backend.Run in order. A migration recorded by the backend but missing
// from the loaded sources blocks execution plans so down and redo operations do
// not operate from incomplete source history.
//
// Backend implementations connect this package to a real store. The
// golazy.dev/pg/pgmigrate package implements Backend for PostgreSQL. It parses
// SQL files with -- +lazy Up and -- +lazy Down sections, stores applied IDs in
// lazy_migrations, runs each step in a transaction, and uses an advisory lock
// while applying a migration. lazymigrate itself does not parse those sections;
// it passes the file content through unchanged so other backends can use their
// own format.
//
// The fakemigrator subpackage provides an in-memory Backend for tests,
// examples, and early command wiring. It records planned steps and applied IDs
// but does not execute SQL.
package lazymigrate
