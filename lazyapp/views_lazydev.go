//go:build lazydev

package lazyapp

import (
	"io/fs"
	"os"
	"strings"

	"golazy.dev/lazyassets"
)

var ViewsPath = "app/views"
var PublicPath = "app/public"

func openConfiguredViews(_ func() (fs.FS, error)) (fs.FS, error) {
	return os.DirFS(lazyDevViewsPath()), nil
}

func openConfiguredPublic(_ func() (fs.FS, error)) (fs.FS, error) {
	return os.DirFS(lazyDevPublicPath()), nil
}

func lazyDevAssetOptions() []lazyassets.Option {
	return []lazyassets.Option{lazyassets.WithDevelopmentMode(true)}
}

func lazyDevViewsPath() string {
	path := strings.TrimSpace(ViewsPath)
	if path == "" {
		return "app/views"
	}
	return path
}

func lazyDevPublicPath() string {
	path := strings.TrimSpace(PublicPath)
	if path == "" {
		return "app/public"
	}
	return path
}
