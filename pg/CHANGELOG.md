# Changelog

## [Unreleased]

- Added `pgauth`, a PostgreSQL `lazyauth.Authenticator` with embedded user
  table migrations and helper methods for creating password-backed users.

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
