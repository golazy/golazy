//go:build lazydev

package lazyapp

import (
	"io/fs"

	"golazy.dev/lazyviews"
)

func openConfiguredViews(_ func() (fs.FS, error)) (fs.FS, error) {
	return lazyviews.Open()
}
