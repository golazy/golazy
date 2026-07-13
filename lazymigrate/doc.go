// Package lazymigrate loads, plans, and applies ordered file-based migrations
// through backend-owned execution.
//
// The package is intentionally backend-agnostic. It knows how to collect source
// migrations, compare them with the migration IDs a backend reports as already
// applied, and build an up, down, or redo plan. The backend owns the concrete
// migration language, the meaning of up and down sections, locking,
// transactions, metadata tables, and schema load or dump support. Production
// backend implementations must synchronize migration application across
// processes so multiple app instances can start with migration mode enabled
// without applying the same step twice or corrupting migration metadata.
//
// A Source returns migration files. FromFS adapts an fs.FS plus an explicit
// directory, skips nested directories and migrations.toml, rejects Go migration
// files, and requires each filename to contain a sortable timestamp. ForDatabase
// is the conventional app-root helper for db/<database>/migrations, for example
// db/postgres/migrations. File extensions are ignored for migration IDs:
// 202606280001_create_documents.sql becomes ID 202606280001_create_documents.
// Prefix and Timestamp are parsed only for stable ordering; duplicate IDs
// across all loaded sources are rejected.
//
// Catalog is useful when an application combines its own migrations with
// package-provided migrations. Add accepts an already adapted Source. Mount
// places an fs.FS beneath a stable namespace in a lazyfs stack, allowing
// independently initialized add-ons to contribute migration trees before the
// catalog is loaded. PostgreSQL service packages in golazy.dev/pg, such as
// pgjobs, pgfiles, pgmedia, and pgstorage, expose embedded migrations as
// lazymigrate.Source values so an application can add them to the same catalog
// as application migrations.
//
// DB and Databases describe the same source/backend relationship for lazyapp
// integration. Each Databases entry is one logical database with its own Backend
// and Source values. This keeps lazymigrate backend-agnostic while letting a
// conventional lazyapp bundle the migrations needed by the application binary.
//
// A Migrator is the planner and executor. List reports applied source
// migrations, pending source migrations, and missing backend migrations.
// PlanUp, PlanDown, and PlanRedo return the Step values that would run without
// setting up backend metadata. Up, Down, and Redo call Backend.Setup before
// planning, then pass each Step to Backend.Run in order. Apply is for callers
// that already have a Plan; it also calls Backend.Setup before running that
// plan. A migration recorded by the backend but missing from the loaded sources
// blocks execution plans so down and redo operations do not operate from
// incomplete source history.
//
// Backend implementations connect this package to a real store. The
// golazy.dev/pg/pgmigrate package implements Backend for PostgreSQL. It parses
// SQL files with -- +lazy Up and -- +lazy Down sections, stores applied IDs in
// lazy_migrations, runs each step and metadata update in a transaction, and
// uses an advisory lock while applying a migration. lazymigrate itself does not
// parse those sections; it passes the file content through unchanged so other
// backends can use their own format.
//
// The fakemigrator subpackage provides an in-memory Backend for tests,
// examples, and early command wiring. It records planned steps and applied IDs
// but does not execute SQL.
package lazymigrate
