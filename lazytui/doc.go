// Package lazytui groups terminal UI helpers for GoLazy command-line tools.
//
// The root package is intentionally only a discovery point. Concrete behavior
// lives in subpackages: progress owns multi-step command status UIs, pty wraps
// pseudo-terminal command execution, window describes terminal dimensions, and
// encoding packages parse or emit terminal byte streams.
//
// GoLazy's lazy command uses these packages for visible setup, build,
// migration, and foreground-process workflows. Applications normally do not
// import lazytui unless they are building their own terminal tooling.
package lazytui
