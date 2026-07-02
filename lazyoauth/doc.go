// Package lazyoauth provides OAuth server and resource-server primitives.
//
// It is intentionally independent from specific account databases and from
// MCP. Applications provide lazyauth authenticators, token stores, and claim
// mapping policy. lazyapp can then use the same OAuth server for MCP clients
// such as Codex or Claude, browser-integrated companion applications, or other
// dependent clients.
package lazyoauth
