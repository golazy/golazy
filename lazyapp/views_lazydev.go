//go:build lazydev

package lazyapp

import (
	"fmt"
	"io/fs"
	"os"
	"strings"
)

const lazyDevViewsPathEnv = "GOLAZY_VIEW_PATH"

func openConfiguredViews(_ func() (fs.FS, error)) (fs.FS, error) {
	viewPath := strings.TrimSpace(os.Getenv(lazyDevViewsPathEnv))
	if viewPath == "" {
		viewPath = "app/views"
	}

	views := os.DirFS(viewPath)
	if _, err := fs.Stat(views, "layouts/app.html.tpl"); err != nil {
		return nil, fmt.Errorf("lazydev views path %q: %w", viewPath, err)
	}
	return views, nil
}
