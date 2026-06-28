//go:build !lazydev

package lazytelemetry

import "log/slog"

func defaultMiddlewareLogger(config Config) *slog.Logger {
	return NewLogger(config, nil)
}
