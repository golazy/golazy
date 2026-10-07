package lazymcp

import (
	"context"
	"strings"

	"golazy.dev/lazyview"
)

// NewViewRenderer adapts an initialized lazyview renderer for MCP App resources.
// It can be used with a standalone Scope without starting a lazyapp server.
func NewViewRenderer(views *lazyview.Views) Views { return viewRenderer{views: views} }

type viewRenderer struct {
	views *lazyview.Views
}

func (renderer viewRenderer) RenderString(ctx context.Context, options ViewOptions) (string, error) {
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
