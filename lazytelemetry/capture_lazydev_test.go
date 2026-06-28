//go:build lazydev

package lazytelemetry

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golazy.dev/lazytelemetry/lazylogs"
	"golazy.dev/lazytelemetry/lazymetrics"
)

func TestMiddlewareWritesRequestCaptureFilesWhenMonitoringEnabled(t *testing.T) {
	t.Chdir(t.TempDir())
	SetRequestMonitoringEnabled(true)
	t.Cleanup(func() {
		SetRequestMonitoringEnabled(false)
	})

	var logs bytes.Buffer
	middleware := MiddlewareFromConfig(
		Config{},
		WithMiddlewareLogger(slog.New(slog.NewJSONHandler(&logs, nil))),
		WithMetricsRegistry(lazymetrics.NewRegistry()),
	)
	handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lazylogs.Info(r.Context(), "inside handler", slog.String("handler", "ok"))
		_, _ = w.Write([]byte("ok"))
	}))
	request := httptest.NewRequest(http.MethodGet, "/articles", nil)
	request.Header.Set("X-Request-ID", "req-123")
	request.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	tracePath := filepath.Join(".tmp", "traces", "req-123.trace")
	spansPath := filepath.Join(".tmp", "traces", "req-123.spans")
	logPath := filepath.Join(".tmp", "traces", "req-123.log.json")
	info, err := os.Stat(tracePath)
	if err != nil {
		t.Fatalf("stat trace file: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("trace file is empty")
	}

	spansData, err := os.ReadFile(spansPath)
	if err != nil {
		t.Fatalf("read spans file: %v", err)
	}
	var spans struct {
		RequestID string `json:"request_id"`
		TraceFile string `json:"trace_file"`
		Runtime   struct {
			GoVersion string `json:"go_version"`
		} `json:"runtime"`
		Memory struct {
			MallocsDelta uint64 `json:"mallocs_delta"`
		} `json:"memory"`
		Spans []struct {
			Name     string `json:"name"`
			TraceID  string `json:"trace_id"`
			ParentID string `json:"parent_id"`
			Events   []struct {
				Name       string                 `json:"name"`
				Attributes map[string]interface{} `json:"attributes"`
			} `json:"events"`
		} `json:"spans"`
	}
	if err := json.Unmarshal(spansData, &spans); err != nil {
		t.Fatalf("parse spans file: %v\n%s", err, spansData)
	}
	if spans.RequestID != "req-123" {
		t.Fatalf("request_id = %q, want req-123", spans.RequestID)
	}
	if spans.TraceFile != ".tmp/traces/req-123.trace" {
		t.Fatalf("trace_file = %q", spans.TraceFile)
	}
	if spans.Runtime.GoVersion == "" {
		t.Fatal("runtime.go_version is empty")
	}
	if len(spans.Spans) != 1 {
		t.Fatalf("spans = %#v", spans.Spans)
	}
	if spans.Spans[0].Name != "http.server.request" {
		t.Fatalf("span name = %q", spans.Spans[0].Name)
	}
	if spans.Spans[0].TraceID != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Fatalf("trace_id = %q", spans.Spans[0].TraceID)
	}
	if spans.Spans[0].ParentID != "00f067aa0ba902b7" {
		t.Fatalf("parent_id = %q", spans.Spans[0].ParentID)
	}
	if !spanEventsContainMessage(spans.Spans[0].Events, "inside handler") {
		t.Fatalf("span events = %#v, want inside handler log", spans.Spans[0].Events)
	}
	if !spanEventsContainMessage(spans.Spans[0].Events, "request completed") {
		t.Fatalf("span events = %#v, want request completed log", spans.Spans[0].Events)
	}

	logData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	logOutput := string(logData)
	for _, want := range []string{
		`"request_id":"req-123"`,
		`"trace_id":"4bf92f3577b34da6a3ce929d0e0e4736"`,
		`"message":"inside handler"`,
		`"message":"request completed"`,
	} {
		if !strings.Contains(logOutput, want) {
			t.Fatalf("log file %q does not contain %q", logOutput, want)
		}
	}
}

func TestMiddlewareSkipsRequestCaptureFilesWhenMonitoringDisabled(t *testing.T) {
	t.Chdir(t.TempDir())
	SetRequestMonitoringEnabled(false)

	middleware := MiddlewareFromConfig(
		Config{TracesExporter: []string{"otlp"}, LogsExporter: []string{"otlp"}},
		WithMiddlewareLogger(slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil))),
		WithMetricsRegistry(lazymetrics.NewRegistry()),
	)
	handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-Request-ID", "req-123")

	handler.ServeHTTP(httptest.NewRecorder(), request)

	if _, err := os.Stat(filepath.Join(".tmp", "traces", "req-123.spans")); !os.IsNotExist(err) {
		t.Fatalf("spans file exists while monitoring is disabled: %v", err)
	}
}

func spanEventsContainMessage(events []struct {
	Name       string                 `json:"name"`
	Attributes map[string]interface{} `json:"attributes"`
}, message string) bool {
	for _, event := range events {
		if event.Name == "log" && event.Attributes["message"] == message {
			return true
		}
	}
	return false
}
