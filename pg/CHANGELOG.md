# Changelog

## [Unreleased]

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
