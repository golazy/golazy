package lazytelemetry

import "sync/atomic"

const requestCaptureDirectory = ".tmp/traces"

// LazyDevRequestMonitoringPath is the lazydev control-plane endpoint for
// detailed request monitoring state.
const LazyDevRequestMonitoringPath = "/requests/monitoring"

// LazyDevRequestMonitoringOnPath enables detailed lazydev request monitoring.
const LazyDevRequestMonitoringOnPath = "/requests/monitoring/on"

// LazyDevRequestMonitoringOffPath disables detailed lazydev request monitoring.
const LazyDevRequestMonitoringOffPath = "/requests/monitoring/off"

// LazyDevRequestTracesPath is the lazydev control-plane endpoint for recorded
// request trace summaries.
const LazyDevRequestTracesPath = "/requests/traces"

var requestMonitoringEnabled atomic.Bool

// RequestMonitoringSnapshot describes detailed lazydev request monitoring.
type RequestMonitoringSnapshot struct {
	Enabled   bool   `json:"enabled"`
	Directory string `json:"directory"`
}

// SetRequestMonitoringEnabled switches detailed request monitoring on or off.
func SetRequestMonitoringEnabled(enabled bool) {
	requestMonitoringEnabled.Store(enabled)
}

// RequestMonitoringEnabled reports whether detailed request monitoring is on.
func RequestMonitoringEnabled() bool {
	return requestMonitoringEnabled.Load()
}

// RequestMonitoringState returns the current detailed request monitoring state.
func RequestMonitoringState() RequestMonitoringSnapshot {
	return RequestMonitoringSnapshot{
		Enabled:   RequestMonitoringEnabled(),
		Directory: requestCaptureDirectory,
	}
}
