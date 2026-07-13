// Package postgres registers the base PostgreSQL add-on for GoLazy apps.
//
// Importing the package makes the "postgres" add-on available to
// lazyaddon.Selection. The add-on opens the application PostgreSQL pool during
// dependency initialization, publishes it as a typed add-on capability, and
// configures the PostgreSQL migration backend.
package postgres
