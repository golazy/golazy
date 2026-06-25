//go:build !lazydev

package lazyapp

import "golazy.dev/lazyassets"

func lazyDevAssetOptions() []lazyassets.Option {
	return nil
}
