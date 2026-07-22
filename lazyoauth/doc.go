// Package lazyoauth provides OAuth server and resource-server primitives.
//
// It is intentionally independent from specific account databases and from
// MCP. Applications provide lazyauth authenticators, token stores, and claim
// mapping policy. Signed refresh tokens carry their grant across server
// restarts while Config.RefreshTokenStates accepts a focused durable
// RefreshTokenStateStore for rotation, replay detection, and family
// revocation. lazyapp can then use the same OAuth server for MCP clients such
// as Codex or Claude, browser-integrated companion applications, or other
// dependent clients.
package lazyoauth
