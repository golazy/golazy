//go:build !lazydev

package lazyapp

import "io/fs"

func openConfiguredViews(open func() (fs.FS, error)) (fs.FS, error) {
	return open()
}
