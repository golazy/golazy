//go:build lazydev

package lazyapp

import "golazy.dev/lazytelemetry"

func telemetryMiddlewareEnabled(config lazytelemetry.Config) bool {
	return !config.SDKDisabled
}
