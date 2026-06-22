// Package lazyview provides view rendering, helper registration, and render
// context handling independent from any concrete template engine.
//
// lazyapp creates a lazyview.Views value for conventional applications and
// lazycontroller renders through it. Template engines are registered by
// importing subpackages such as golazy.dev/lazyview/gotmpl.
//
// Use lazyview directly for custom rendering stacks, non-controller output, or
// packages such as lazymailer that share the same embedded views and helpers
// without serving an HTTP request.
package lazyview
