// Package lazyjwt signs and validates JSON Web Tokens for GoLazy packages.
//
// The package intentionally stays below OAuth, MCP, and application account
// systems. It owns token encoding, signature verification, registered claim
// validation, extra claim accessors, and request-context helpers for packages
// that receive an already validated token.
package lazyjwt
