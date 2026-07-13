# Changelog

## [Unreleased]

- Added the `postgres` and dependent `postgres/jobs` add-on packages. The base
  add-on registers the shared pgx pool, migration backend, and lazydev database
  panel with a safe database-ping action; the jobs add-on mounts its migrations
  and supplies the durable jobs backend only when selected.
- The `postgres` add-on accepts committed `database_url_variable`
  configuration so applications can name the environment variable containing
  their secret connection URL without storing the URL in `addons.toml`.
- Added `pgjobs.MigrationFiles` for mounting the embedded jobs migration
  filesystem into a shared `lazymigrate.Catalog`.
- Added `pgauth`, a PostgreSQL `lazyauth.Authenticator` with embedded user
  table migrations and helper methods for creating password-backed users.
- `pgjobs` now persists fixed-interval schedules, records job `schedule_key`,
  and enforces `lazyjobs.Config.QueueLimits` atomically while claiming jobs.

## [0.1.19] - 2026-07-03

- `pgmigrate` now treats stale concurrent up/down steps for an already-applied
  migration with the same checksum as successful no-ops, while still rejecting
  the same migration ID when the stored checksum differs.

## [0.1.18] - 2026-06-30

- Added `WithPool` and `FromContext` helpers for sharing an app-owned
  `pgxpool.Pool` through the GoLazy application context.
- Added PostgreSQL implementations for `lazyfiles`, `lazymedia`, and
  `lazystorage` with embedded migrations under `pgfiles`, `pgmedia`, and
  `pgstorage`.

## [0.1.17] - 2026-06-29

- Added the first `golazy.dev/pg` module with PostgreSQL migration and job
  backend packages.
- Removed reserved placeholder packages for assets, files, migrations, and
  storage until those integrations have concrete implementations.
