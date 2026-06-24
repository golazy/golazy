package lazycontroller

import (
	"embed"
	"fmt"
	"io/fs"
)

//go:embed app/views/layouts/app.html.tpl app/views/app/error.html.tpl
var defaultViewFiles embed.FS

// DefaultViews returns the framework-owned fallback view filesystem.
//
// lazyapp layers application views over these files so apps can override
// layouts/app.html.tpl or app/error.html.tpl with ordinary view files.
func DefaultViews() (fs.FS, error) {
	views, err := fs.Sub(defaultViewFiles, "app/views")
	if err != nil {
		return nil, fmt.Errorf("lazycontroller: open default views: %w", err)
	}
	return views, nil
}
