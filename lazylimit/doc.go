// Package lazylimit provides generic rate limiting primitives.
//
// Limiters are keyed by caller-provided strings, so the same package can back
// HTTP middleware, OAuth endpoints, MCP tools, or application services. The
// in-memory limiter is intended as a development and single-process backend;
// applications can provide another Store-compatible limiter for distributed
// deployments.
package lazylimit
