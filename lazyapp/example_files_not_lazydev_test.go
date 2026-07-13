//go:build !lazydev

package lazyapp_test

import "io/fs"

func configureExampleFiles(fs.FS, fs.FS) func() { return func() {} }
