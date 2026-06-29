// Package pg contains PostgreSQL helpers shared by concrete GoLazy
// PostgreSQL backends and applications.
//
// The package is intentionally small: it stores an application-owned
// pgxpool.Pool on context with WithPool and retrieves it with FromContext. That
// keeps database ownership in the application while allowing backend packages
// such as pgmigrate, pgjobs, and other pg/* implementations to share the same
// connection pool.
//
// lazyapp does not create a PostgreSQL pool by default. Applications normally
// create the pool in lazyapp.Config.Dependencies, attach it with WithPool, and
// let database-backed services or jobs read it back from context.
package pg
