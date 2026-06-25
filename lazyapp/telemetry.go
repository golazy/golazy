package lazyapp

import (
	"golazy.dev/lazycache"
	"golazy.dev/lazycontrolplane"
	"golazy.dev/lazytelemetry"
	"golazy.dev/lazytelemetry/lazymetrics"
)

func telemetryControlPlane(controlPlane *lazycontrolplane.ControlPlane, config lazytelemetry.Config, registry *lazymetrics.Registry, cache *lazycache.Cache) *lazycontrolplane.ControlPlane {
	if !config.PrometheusMetrics() {
		return controlPlane
	}
	if controlPlane == nil {
		controlPlane = lazycontrolplane.New(lazycontrolplane.Config{})
	}
	if controlPlane.HandlesPath("/metrics") {
		return controlPlane
	}
	controlPlane.Handle("GET /metrics", lazymetrics.PrometheusHandler(registry, lazycache.PrometheusCollector(cache)))
	return controlPlane
}
