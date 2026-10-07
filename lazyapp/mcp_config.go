package lazyapp

import (
	"context"

	"golazy.dev/lazyauth"
	"golazy.dev/lazymcp"
	"golazy.dev/lazyoauth"
)

// AuthConfig initializes lazyauth with the dependency-initialized app context.
type AuthConfig func(context.Context) (lazyauth.Config, error)

// Auth adapts a static lazyauth.Config for Config.Auth.
func Auth(config lazyauth.Config) AuthConfig {
	return func(context.Context) (lazyauth.Config, error) {
		return config, nil
	}
}

// OAuthConfig initializes lazyoauth with the dependency-initialized app
// context. Applications can resolve a durable RefreshTokenStateStore from this
// context and assign it to lazyoauth.Config.RefreshTokenStates.
type OAuthConfig func(context.Context) (lazyoauth.Config, error)

// OAuth adapts a static lazyoauth.Config for Config.OAuth.
func OAuth(config lazyoauth.Config) OAuthConfig {
	return func(context.Context) (lazyoauth.Config, error) {
		return config, nil
	}
}

// MCPConfig registers application MCP modules.
type MCPConfig func(context.Context, *lazymcp.Scope) error
