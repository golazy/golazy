//go:build lazydev

package lazytelemetry

import (
	"io"
	"log/slog"
)

func defaultMiddlewareLogger(config Config) *slog.Logger {
	if config.Enabled() {
		return NewLogger(config, nil)
	}
	return NewLogger(config, io.Discard)
}
