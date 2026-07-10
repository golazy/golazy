// Package lazyauth authenticates users without owning application
// authorization.
//
// Authenticators can validate passwords, magic links, bearer tokens, OAuth
// provider callbacks, or application-specific credentials. Successful
// authentication returns a User with an ID and serializable Data map. Packages
// above lazyauth decide how that data becomes OAuth claims, sessions, roles, or
// MCP permissions.
//
// The memoryauth subpackage provides the default lazyapp backend. It starts
// with zero users unless LAZYAUTH_DEFAULT_PASS creates a bootstrap password
// user named admin, or LAZYAUTH_DEFAULT_USER when that value is set.
package lazyauth
