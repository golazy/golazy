// Package lazymcp exposes GoLazy application capabilities through the Model
// Context Protocol.
//
// Applications normally configure MCP through lazyapp.Config.MCP. The package
// can also be used directly by constructing a Scope, registering MCP modules,
// and mounting the scope as an HTTP handler or calling Scope.ServeStdio for a
// local subprocess. NewViewRenderer adapts standalone lazyview rendering for
// MCP Apps. Keep stdout exclusively for protocol messages and logs on stderr.
// Authentication and authorization
// are intentionally supplied by outer layers; lazymcp reads validated JWT
// claims from context when they are present and filters module access from the
// "mcps" claim by default.
package lazymcp
