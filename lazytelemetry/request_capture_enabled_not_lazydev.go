//go:build !lazydev

package lazytelemetry

func captureRequestFilesEnabled(config Config) bool {
	return config.captureRequestFiles()
}
