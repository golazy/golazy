//go:build lazydev

package lazytelemetry

func captureRequestFilesEnabled(Config) bool {
	return RequestMonitoringEnabled()
}
