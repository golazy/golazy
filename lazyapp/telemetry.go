package lazyapp

import (
	"context"
	"time"

	"golazy.dev/lazycache"
	"golazy.dev/lazycontrolplane"
	"golazy.dev/lazydeps"
	"golazy.dev/lazytelemetry"
)

const telemetryShutdownTimeout = 5 * time.Second

func initializeTelemetry(dependencies *lazydeps.Scope) (*lazytelemetry.Telemetry, error) {
	var telemetry *lazytelemetry.Telemetry
	_, err := lazydeps.Service(dependencies, "telemetry", func(ctx context.Context) (context.Context, *lazytelemetry.Telemetry, error, context.CancelFunc) {
		instance, nextCtx, err := lazytelemetry.New(ctx)
		if err != nil {
			return ctx, nil, err, nil
		}
		telemetry = instance
		stop := func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), telemetryShutdownTimeout)
			defer cancel()
			if err := instance.Shutdown(shutdownCtx); err != nil {
				instance.Logger().Error("lazyapp: shutdown telemetry", "err", err)
			}
		}
		return nextCtx, instance, nil, stop
	})
	if err != nil {
		return nil, err
	}
	return telemetry, nil
}

func telemetryControlPlane(controlPlane *lazycontrolplane.ControlPlane, telemetry *lazytelemetry.Telemetry, cache *lazycache.Cache) *lazycontrolplane.ControlPlane {
	if telemetry == nil || !telemetry.Config().PrometheusMetrics() {
		return controlPlane
	}
	if controlPlane == nil {
		controlPlane = lazycontrolplane.New(lazycontrolplane.Config{})
	}
	if controlPlane.HandlesPath("/metrics") {
		return controlPlane
	}
	controlPlane.Handle("GET /metrics", telemetry.PrometheusHandler(lazycache.PrometheusCollector(cache)))
	return controlPlane
}
