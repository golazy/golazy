// Package pgmigrate implements PostgreSQL migrations for golazy.dev/lazymigrate.
//
// Migration files are self-contained SQL files with lazy markers:
//
//	-- +lazy Up
//	CREATE TABLE example (...);
//
//	-- +lazy Down
//	DROP TABLE example;
package pgmigrate
