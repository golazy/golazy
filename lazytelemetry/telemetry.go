package lazytelemetry

import (
	"context"
	"log/slog"
	"net/http"
	"sync"

	"go.opentelemetry.io/contrib/exporters/autoexport"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"golazy.dev/lazydispatch"
	"golazy.dev/lazytelemetry/lazymetrics"
)

type telemetryContextKey struct{}

// Telemetry owns the app telemetry runtime initialized from OTEL_* settings.
type Telemetry struct {
	config         Config
	logger         *slog.Logger
	registry       *lazymetrics.Registry
	tracerProvider *sdktrace.TracerProvider

	shutdownOnce sync.Once
	shutdownErr  error
}

// New initializes telemetry from OTEL_* environment variables and attaches it
// to the returned context.
func New(ctx context.Context) (*Telemetry, context.Context, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, ctx, err
	}
	return newTelemetry(ctx, config)
}

func newTelemetry(ctx context.Context, config Config) (*Telemetry, context.Context, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	telemetry := &Telemetry{
		config:   config,
		logger:   defaultMiddlewareLogger(config),
		registry: lazymetrics.NewRegistry(),
	}
	if config.traceExportEnabled() {
		provider, err := newTracerProvider(ctx)
		if err != nil {
			return nil, ctx, err
		}
		telemetry.tracerProvider = provider
	}
	return telemetry, WithTelemetry(ctx, telemetry), nil
}

func newTracerProvider(ctx context.Context) (*sdktrace.TracerProvider, error) {
	exporter, err := autoexport.NewSpanExporter(ctx)
	if err != nil {
		return nil, err
	}
	if autoexport.IsNoneSpanExporter(exporter) {
		return nil, nil
	}
	res, err := resource.New(ctx,
		resource.WithService(),
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
	)
	if err != nil {
		return nil, err
	}
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(provider)
	return provider, nil
}

// WithTelemetry attaches telemetry to ctx.
func WithTelemetry(ctx context.Context, telemetry *Telemetry) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if telemetry == nil {
		return ctx
	}
	return context.WithValue(ctx, telemetryContextKey{}, telemetry)
}

// FromContext returns the telemetry instance attached to ctx.
func FromContext(ctx context.Context) (*Telemetry, bool) {
	if ctx == nil {
		return nil, false
	}
	telemetry, ok := ctx.Value(telemetryContextKey{}).(*Telemetry)
	return telemetry, ok && telemetry != nil
}

// Config returns the OTEL_* configuration used to initialize telemetry.
func (telemetry *Telemetry) Config() Config {
	if telemetry == nil {
		return Config{}
	}
	return telemetry.config
}

// Enabled reports whether any telemetry setting is active.
func (telemetry *Telemetry) Enabled() bool {
	return telemetry != nil && telemetry.config.Enabled()
}

// Logger returns the logger used by telemetry middleware.
func (telemetry *Telemetry) Logger() *slog.Logger {
	if telemetry == nil || telemetry.logger == nil {
		return slog.Default()
	}
	return telemetry.logger
}

// MetricsRegistry returns the in-process metrics registry.
func (telemetry *Telemetry) MetricsRegistry() *lazymetrics.Registry {
	if telemetry == nil || telemetry.registry == nil {
		return lazymetrics.NewRegistry()
	}
	return telemetry.registry
}

// Middleware returns the request telemetry middleware for this instance.
func (telemetry *Telemetry) Middleware() lazydispatch.Middleware {
	if telemetry == nil {
		return nil
	}
	return MiddlewareFromConfig(
		telemetry.config,
		WithMiddlewareLogger(telemetry.Logger()),
		WithMetricsRegistry(telemetry.MetricsRegistry()),
	)
}

// PrometheusHandler returns a Prometheus exposition handler for this instance.
func (telemetry *Telemetry) PrometheusHandler(collectors ...lazymetrics.PrometheusCollector) http.Handler {
	return lazymetrics.PrometheusHandler(telemetry.MetricsRegistry(), collectors...)
}

// Shutdown flushes and stops telemetry exporters.
func (telemetry *Telemetry) Shutdown(ctx context.Context) error {
	if telemetry == nil {
		return nil
	}
	telemetry.shutdownOnce.Do(func() {
		if telemetry.tracerProvider != nil {
			telemetry.shutdownErr = telemetry.tracerProvider.Shutdown(ctx)
		}
	})
	return telemetry.shutdownErr
}
