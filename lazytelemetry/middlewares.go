package lazytelemetry

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golazy.dev/lazydispatch"
	"golazy.dev/lazytelemetry/lazylogs"
	"golazy.dev/lazytelemetry/lazymetrics"
	"golazy.dev/lazytelemetry/lazytracing"
)

const requestIDHeader = "X-Request-ID"

// MiddlewareOption configures the telemetry middleware.
type MiddlewareOption func(*middleware)

type middleware struct {
	logger   *slog.Logger
	registry *lazymetrics.Registry

	requestsTotal   *lazymetrics.CounterVec
	requestDuration *lazymetrics.HistogramVec
}

// Middleware returns the default telemetry middleware.
func Middleware(options ...MiddlewareOption) lazydispatch.Middleware {
	return MiddlewareFromConfig(MustLoadConfig(), options...)
}

// EnvironmentMiddleware returns a middleware when environment configuration
// activates telemetry.
func EnvironmentMiddleware(options ...MiddlewareOption) (lazydispatch.Middleware, bool) {
	config := MustLoadConfig()
	if !config.Enabled() {
		return nil, false
	}
	return MiddlewareFromConfig(config, options...), true
}

// MiddlewareFromConfig returns a middleware configured from config.
func MiddlewareFromConfig(config Config, options ...MiddlewareOption) lazydispatch.Middleware {
	middleware := &middleware{
		logger:   NewLogger(config, nil),
		registry: lazymetrics.NewRegistry(),
	}
	for _, option := range options {
		if option != nil {
			option(middleware)
		}
	}
	if middleware.logger == nil {
		middleware.logger = NewLogger(config, nil)
	}
	if middleware.registry == nil {
		middleware.registry = lazymetrics.NewRegistry()
	}
	middleware.requestsTotal = middleware.registry.NewCounter("http_server_requests_total", "method", "route", "status_class")
	middleware.requestDuration = middleware.registry.NewHistogram("http_server_request_duration_seconds", "method", "route", "status_class")
	return middleware
}

// WithMiddlewareLogger configures the logger attached to request contexts.
func WithMiddlewareLogger(logger *slog.Logger) MiddlewareOption {
	return func(middleware *middleware) {
		middleware.logger = logger
	}
}

// WithMetricsRegistry configures the registry used by request metrics.
func WithMetricsRegistry(registry *lazymetrics.Registry) MiddlewareOption {
	return func(middleware *middleware) {
		middleware.registry = registry
	}
}

// Handler implements lazydispatch.Middleware.
func (middleware *middleware) Handler(next http.Handler) http.Handler {
	if next == nil {
		next = http.NotFoundHandler()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		requestID := requestIDFromHeaders(r.Header)
		if requestID == "" {
			requestID = generateRequestID()
		}

		ctx := WithRequestID(r.Context(), requestID)
		ctx = lazylogs.WithLogger(ctx, middleware.logger)
		ctx = lazylogs.WithAttrs(ctx,
			slog.String("request_id", requestID),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
		)
		ctx = lazymetrics.WithLabels(ctx, lazymetrics.Labels{
			"method": r.Method,
			"route":  "unknown",
		})
		if traceContext, ok := lazytracing.ParseTraceparent(r.Header.Get("traceparent"), r.Header.Get("tracestate")); ok {
			ctx = lazytracing.WithTraceContext(ctx, traceContext)
		}

		ctx, span := lazytracing.StartSpan(ctx, "http.server.request",
			slog.String("http.request.method", r.Method),
			slog.String("url.path", r.URL.Path),
			slog.String("request_id", requestID),
		)
		ctx = lazylogs.WithAttrs(ctx,
			slog.String("trace_id", span.TraceID()),
			slog.String("span_id", span.SpanID()),
		)

		w.Header().Set(requestIDHeader, requestID)
		recorder := &responseRecorder{ResponseWriter: w}
		request := r.WithContext(ctx)

		defer func() {
			recovered := recover()
			status := recorder.Status()
			duration := time.Since(startedAt)
			labels := lazymetrics.LabelsFromContext(ctx)
			labels["status_class"] = statusClass(status)

			middleware.requestsTotal.With(labels).Inc()
			middleware.requestDuration.With(labels).Observe(duration.Seconds())
			span.AddAttributes(
				slog.Int("http.response.status_code", status),
				slog.Duration("duration", duration),
			)
			if recovered != nil {
				err := fmt.Errorf("panic: %v", recovered)
				span.RecordError(err)
				lazylogs.Error(ctx, "request panicked",
					slog.Int("status", status),
					slog.Duration("duration", duration),
					slog.Any("panic", recovered),
				)
				span.End()
				panic(recovered)
			}
			lazylogs.Info(ctx, "request completed",
				slog.Int("status", status),
				slog.Duration("duration", duration),
				slog.Int("bytes", recorder.BytesWritten()),
			)
			span.End()
		}()

		next.ServeHTTP(recorder, request)
	})
}

func requestIDFromHeaders(header http.Header) string {
	for _, name := range []string{requestIDHeader, "X-Correlation-ID"} {
		for _, value := range header.Values(name) {
			if validRequestID(value) {
				return strings.TrimSpace(value)
			}
		}
	}
	return ""
}

func validRequestID(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 128 || strings.Contains(value, ",") {
		return false
	}
	for _, char := range value {
		if char >= 'a' && char <= 'z' ||
			char >= 'A' && char <= 'Z' ||
			char >= '0' && char <= '9' ||
			char == '_' || char == '-' || char == '.' || char == ':' || char == '/' {
			continue
		}
		return false
	}
	return true
}

func generateRequestID() string {
	data := make([]byte, 16)
	if _, err := rand.Read(data); err != nil {
		panic(fmt.Errorf("lazytelemetry: generate request id: %w", err))
	}
	return base64.RawURLEncoding.EncodeToString(data)
}

func statusClass(status int) string {
	if status < 100 {
		status = http.StatusOK
	}
	return strconv.Itoa(status/100) + "xx"
}

type responseRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *responseRecorder) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseRecorder) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(data)
	w.bytes += n
	return n, err
}

func (w *responseRecorder) Status() int {
	if w.status == 0 {
		return http.StatusOK
	}
	return w.status
}

func (w *responseRecorder) BytesWritten() int {
	return w.bytes
}

func (w *responseRecorder) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *responseRecorder) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		if w.status == 0 {
			w.WriteHeader(http.StatusOK)
		}
		flusher.Flush()
	}
}
