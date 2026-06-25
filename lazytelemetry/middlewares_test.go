package lazytelemetry

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golazy.dev/lazytelemetry/lazylogs"
	"golazy.dev/lazytelemetry/lazymetrics"
)

func TestMiddlewareAddsRequestIDLoggerSpanAndMetrics(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	registry := lazymetrics.NewRegistry()
	middleware := Middleware(
		WithMiddlewareLogger(logger),
		WithMetricsRegistry(registry),
	)

	handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := RequestID(r.Context()); got != "req-123" {
			t.Fatalf("RequestID = %q", got)
		}
		if got := TraceID(r.Context()); got != "4bf92f3577b34da6a3ce929d0e0e4736" {
			t.Fatalf("TraceID = %q", got)
		}
		if got := SpanID(r.Context()); got == "" || got == "00f067aa0ba902b7" {
			t.Fatalf("SpanID = %q", got)
		}
		lazylogs.Info(r.Context(), "inside handler", slog.String("handler", "ok"))
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("created"))
	}))

	request := httptest.NewRequest(http.MethodPost, "/articles", nil)
	request.Header.Set("X-Request-ID", "req-123")
	request.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d", response.Code)
	}
	if got := response.Header().Get("X-Request-ID"); got != "req-123" {
		t.Fatalf("X-Request-ID = %q", got)
	}
	logOutput := logs.String()
	for _, want := range []string{
		`"request_id":"req-123"`,
		`"trace_id":"4bf92f3577b34da6a3ce929d0e0e4736"`,
		`"msg":"inside handler"`,
		`"msg":"request completed"`,
		`"status":201`,
	} {
		if !strings.Contains(logOutput, want) {
			t.Fatalf("logs %q do not contain %q", logOutput, want)
		}
	}

	snapshot := registry.Snapshot()
	if got := findCounter(snapshot.Counters, "http_server_requests_total", lazymetrics.Labels{
		"method":       "POST",
		"route":        "unknown",
		"status_class": "2xx",
	}); got != 1 {
		t.Fatalf("request counter = %v, want 1", got)
	}
	if got := findHistogramCount(snapshot.Histograms, "http_server_request_duration_seconds", lazymetrics.Labels{
		"method":       "POST",
		"route":        "unknown",
		"status_class": "2xx",
	}); got != 1 {
		t.Fatalf("duration histogram count = %d, want 1", got)
	}
}

func TestMiddlewareGeneratesRequestIDWhenHeaderIsInvalid(t *testing.T) {
	middleware := Middleware(
		WithMiddlewareLogger(slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil))),
		WithMetricsRegistry(lazymetrics.NewRegistry()),
	)
	handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := RequestID(r.Context()); got == "" || got == "bad,id" {
			t.Fatalf("RequestID = %q", got)
		}
	}))

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-Request-ID", "bad,id")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if got := response.Header().Get("X-Request-ID"); got == "" || got == "bad,id" {
		t.Fatalf("X-Request-ID = %q", got)
	}
}

func findCounter(metrics []lazymetrics.MetricSnapshot, name string, labels lazymetrics.Labels) float64 {
	for _, metric := range metrics {
		if metric.Name == name && sameMetricLabels(metric.Labels, labels) {
			return metric.Value
		}
	}
	return 0
}

func findHistogramCount(metrics []lazymetrics.HistogramSnapshot, name string, labels lazymetrics.Labels) int64 {
	for _, metric := range metrics {
		if metric.Name == name && sameMetricLabels(metric.Labels, labels) {
			return metric.Count
		}
	}
	return 0
}

func sameMetricLabels(left, right lazymetrics.Labels) bool {
	if len(left) != len(right) {
		return false
	}
	for name, value := range left {
		if right[name] != value {
			return false
		}
	}
	return true
}
