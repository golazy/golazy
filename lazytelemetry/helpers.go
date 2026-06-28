package lazytelemetry

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"

	"golazy.dev/lazytelemetry/lazylogs"
	"golazy.dev/lazytelemetry/lazymetrics"
	"golazy.dev/lazytelemetry/lazytracing"
)

// WithRequestID attaches requestID to ctx.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return lazytracing.WithRequestID(ctx, requestID)
}

// RequestID returns the request id attached to ctx.
func RequestID(ctx context.Context) string {
	return lazytracing.RequestID(ctx)
}

// Logger returns the slog logger attached to ctx.
func Logger(ctx context.Context) *slog.Logger {
	return lazylogs.Logger(ctx)
}

// WithLogger attaches logger to ctx.
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return lazylogs.WithLogger(ctx, logger)
}

// WithLogAttrs returns a context whose logger includes attrs.
func WithLogAttrs(ctx context.Context, attrs ...slog.Attr) context.Context {
	return lazylogs.WithAttrs(ctx, attrs...)
}

// WithLogTags returns a context whose logger includes tags.
func WithLogTags(ctx context.Context, tags ...string) context.Context {
	return lazylogs.WithTags(ctx, tags...)
}

// WithLogGroup returns a context whose logger writes subsequent attrs in group.
func WithLogGroup(ctx context.Context, group string) context.Context {
	return lazylogs.WithGroup(ctx, group)
}

// StartSpan starts a span and attaches it to the returned context.
func StartSpan(ctx context.Context, name string, attrs ...slog.Attr) (context.Context, *lazytracing.Span) {
	return lazytracing.StartSpan(ctx, name, attrs...)
}

// StartRegion starts a child span and Go runtime trace region when ctx carries
// an active span. It returns ctx unchanged and a nil span when telemetry is not
// active for the request.
func StartRegion(ctx context.Context, name string, attrs ...slog.Attr) (context.Context, *lazytracing.Span) {
	return lazytracing.StartRegion(ctx, name, attrs...)
}

// SpanFromContext returns the active span attached to ctx.
func SpanFromContext(ctx context.Context) *lazytracing.Span {
	return lazytracing.SpanFromContext(ctx)
}

// TraceID returns the trace id attached to ctx.
func TraceID(ctx context.Context) string {
	return lazytracing.TraceID(ctx)
}

// SpanID returns the span id attached to ctx.
func SpanID(ctx context.Context) string {
	return lazytracing.SpanID(ctx)
}

// MetricLabels returns metric labels attached to ctx.
func MetricLabels(ctx context.Context) lazymetrics.Labels {
	return lazymetrics.LabelsFromContext(ctx)
}

// WithMetricLabels attaches metric labels to ctx.
func WithMetricLabels(ctx context.Context, labels lazymetrics.Labels) context.Context {
	return lazymetrics.WithLabels(ctx, labels)
}

// NewLogger builds a default telemetry logger from config.
func NewLogger(config Config, out io.Writer) *slog.Logger {
	if out == nil {
		out = os.Stdout
	}
	if config.JSONLogs() {
		return lazylogs.NewJSONLogger(out)
	}
	return lazylogs.NewTextLogger(out)
}

// JSONLogs reports whether config asks for structured log output.
func (config Config) JSONLogs() bool {
	for _, exporter := range config.LogsExporter {
		exporter = strings.TrimSpace(strings.ToLower(exporter))
		if exporter != "" && exporter != "none" {
			return true
		}
	}
	return false
}
