//go:build !lazydev

package lazyapp

import "testing"

func configureLazyDevViewsForTest(t *testing.T, files map[string]string) {
	t.Helper()
}

func configureLazyDevPublicForTest(t *testing.T, files map[string]string) {
	t.Helper()
}

func lazyDevTestBuild() bool {
	return false
}
