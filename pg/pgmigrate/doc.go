// Package pgmigrate implements PostgreSQL migrations for golazy.dev/lazymigrate.
//
// Migration files are self-contained SQL files with lazy markers:
//
//	-- +lazy Up
//	CREATE TABLE example (...);
//
//	-- +lazy Down
//	DROP TABLE example;
//
// The backend applies each step in a transaction while holding a PostgreSQL
// advisory transaction lock. If a concurrent caller reaches the same migration
// after another process already applied it with the same checksum, the stale
// step succeeds as a no-op. If the stored checksum differs, the backend returns
// an error instead of hiding changed migration content.
package pgmigrate
