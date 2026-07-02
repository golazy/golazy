package lazyapp

import (
	"context"
	"strings"

	"golazy.dev/lazyauth"
	"golazy.dev/lazymcp"
	"golazy.dev/lazyoauth"
	"golazy.dev/lazyview"
)

// AuthConfig initializes lazyauth with the dependency-initialized app context.
type AuthConfig func(context.Context) (lazyauth.Config, error)

// Auth adapts a static lazyauth.Config for Config.Auth.
func Auth(config lazyauth.Config) AuthConfig {
	return func(context.Context) (lazyauth.Config, error) {
		return config, nil
	}
}

// OAuthConfig initializes lazyoauth with the dependency-initialized app context.
type OAuthConfig func(context.Context) (lazyoauth.Config, error)

// OAuth adapts a static lazyoauth.Config for Config.OAuth.
func OAuth(config lazyoauth.Config) OAuthConfig {
	return func(context.Context) (lazyoauth.Config, error) {
		return config, nil
	}
}

// MCPConfig registers application MCP modules.
type MCPConfig func(context.Context, *lazymcp.Scope) error

type mcpViews struct {
	views *lazyview.Views
}

func (renderer mcpViews) RenderString(ctx context.Context, options lazymcp.ViewOptions) (string, error) {
	controller, action := splitMCPView(options.View)
	return renderer.views.RenderString(lazyview.Options{
		Context:    ctx,
		Variables:  options.Variables,
		Controller: controller,
		Action:     action,
		Format:     "html",
		Layout:     options.Layout,
		UseLayout:  options.UseLayout,
	})
}

func splitMCPView(view string) (string, string) {
	view = strings.Trim(view, "/")
	if view == "" {
		return "mcp", "index"
	}
	before, _, ok := strings.Cut(view, "/")
	if !ok {
		return "mcp", before
	}
	parts := strings.Split(view, "/")
	return strings.Join(parts[:len(parts)-1], "/"), parts[len(parts)-1]
}
