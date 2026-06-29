// Package lazybuildinfo exposes Go build metadata to development tooling.
//
// In a normal GoLazy application, lazyapp wires this package into the
// application's lazycontrolplane only for builds compiled with the lazydev build
// tag. The development panel can then request the endpoint to inspect the Go
// toolchain version, main module, dependencies, replacements, and build
// settings reported by runtime/debug.ReadBuildInfo.
//
// Applications rarely need to use this package directly. Use lazyapp when
// building a GoLazy app; it aggregates lazybuildinfo with the other lazydev
// control-plane handlers. Direct registration is useful only for custom
// development servers that create their own lazycontrolplane.ControlPlane.
package lazybuildinfo
