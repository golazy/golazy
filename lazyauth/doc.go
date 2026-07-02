// Package lazyauth authenticates users without owning application
// authorization.
//
// Authenticators can validate passwords, magic links, bearer tokens, OAuth
// provider callbacks, or application-specific credentials. Successful
// authentication returns a User with an ID and serializable Data map. Packages
// above lazyauth decide how that data becomes OAuth claims, sessions, roles, or
// MCP permissions.
package lazyauth
