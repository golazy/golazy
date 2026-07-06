// Package pgauth provides a PostgreSQL-backed lazyauth.Authenticator.
//
// Applications own their pgx pool and pass it to New. Include Migrations() in
// the app's PostgreSQL migration sources before authenticating users.
package pgauth
