//go:build !lazydev

package lazytracing

func currentGoroutineID() uint64 {
	return 0
}
