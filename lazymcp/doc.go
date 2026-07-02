// Package lazymcp exposes GoLazy application capabilities through the Model
// Context Protocol.
//
// Applications normally configure MCP through lazyapp.Config.MCP. The package
// can also be used directly by constructing a Scope, registering MCP modules,
// and mounting the scope as an HTTP handler. Authentication and authorization
// are intentionally supplied by outer layers; lazymcp reads validated JWT
// claims from context when they are present and filters module access from the
// "mcps" claim by default.
package lazymcp
