//go:build lazydev

package lazytelemetry

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golazy.dev/lazytelemetry/lazylogs"
	"golazy.dev/lazytelemetry/lazymetrics"
)

var captureAllocSink [][]byte

func TestMiddlewareWritesRequestCaptureFilesWhenMonitoringEnabled(t *testing.T) {
	t.Chdir(t.TempDir())
	SetRequestMonitoringEnabled(true)
	t.Cleanup(func() {
		SetRequestMonitoringEnabled(false)
		captureAllocSink = nil
	})

	var logs bytes.Buffer
	middleware := MiddlewareFromConfig(
		Config{},
		WithMiddlewareLogger(slog.New(slog.NewJSONHandler(&logs, nil))),
		WithMetricsRegistry(lazymetrics.NewRegistry()),
	)
	handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := StartRegion(r.Context(), "handler.region", slog.String("handler", "ok"))
		if span != nil {
			defer span.End()
		}
		captureAllocSink = append(captureAllocSink, allocateForCaptureTest(256*1024))
		lazylogs.Info(ctx, "inside handler", slog.String("handler", "ok"))
		childCtx, childSpan := StartRegion(ctx, "handler.child")
		captureAllocSink = append(captureAllocSink, allocateForCaptureTest(128*1024))
		lazylogs.Info(childCtx, "inside child")
		if childSpan != nil {
			childSpan.End()
		}
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
			Name           string  `json:"name"`
			TraceID        string  `json:"trace_id"`
			SpanID         string  `json:"span_id"`
			ParentID       string  `json:"parent_id"`
			DurationMS     float64 `json:"duration_ms"`
			SelfDurationMS float64 `json:"self_duration_ms"`
			Memory         *struct {
				TotalAllocBytesDelta     uint64 `json:"total_alloc_bytes_delta"`
				MallocsDelta             uint64 `json:"mallocs_delta"`
				FreesDelta               uint64 `json:"frees_delta"`
				SelfTotalAllocBytesDelta uint64 `json:"self_total_alloc_bytes_delta"`
				SelfMallocsDelta         uint64 `json:"self_mallocs_delta"`
				SelfFreesDelta           uint64 `json:"self_frees_delta"`
			} `json:"memory"`
			Attributes map[string]interface{} `json:"attributes"`
			Events     []struct {
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
	if len(spans.Spans) != 3 {
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
	if !spanEventsContainMessage(spans.Spans[0].Events, "request completed") {
		t.Fatalf("span events = %#v, want request completed log", spans.Spans[0].Events)
	}
	if spans.Spans[1].Name != "handler.region" {
		t.Fatalf("child span name = %q", spans.Spans[1].Name)
	}
	if spans.Spans[1].ParentID != spans.Spans[0].SpanID {
		t.Fatalf("child parent_id = %q, want %q", spans.Spans[1].ParentID, spans.Spans[0].SpanID)
	}
	if spans.Spans[1].Memory == nil {
		t.Fatal("handler span memory is nil")
	}
	if spans.Spans[1].Memory.TotalAllocBytesDelta == 0 || spans.Spans[1].Memory.MallocsDelta == 0 {
		t.Fatalf("handler span memory = %#v, want nonzero allocation sample", spans.Spans[1].Memory)
	}
	if spans.Spans[1].SelfDurationMS < 0 || spans.Spans[1].SelfDurationMS > spans.Spans[1].DurationMS {
		t.Fatalf("handler self duration = %f, duration = %f", spans.Spans[1].SelfDurationMS, spans.Spans[1].DurationMS)
	}
	if spans.Spans[1].Memory.SelfTotalAllocBytesDelta > spans.Spans[1].Memory.TotalAllocBytesDelta {
		t.Fatalf("handler self memory = %#v, want self <= total", spans.Spans[1].Memory)
	}
	if got := spans.Spans[1].Attributes["request_id"]; got != "req-123" {
		t.Fatalf("child request_id attr = %#v, want req-123", got)
	}
	if !spanEventsContainMessage(spans.Spans[1].Events, "inside handler") {
		t.Fatalf("child span events = %#v, want inside handler log", spans.Spans[1].Events)
	}
	if !spanEventsContainAttr(spans.Spans[1].Events, "request_id", "req-123") {
		t.Fatalf("child span events = %#v, want request_id attr", spans.Spans[1].Events)
	}
	if spans.Spans[2].Name != "handler.child" {
		t.Fatalf("grandchild span name = %q", spans.Spans[2].Name)
	}
	if spans.Spans[2].ParentID != spans.Spans[1].SpanID {
		t.Fatalf("grandchild parent_id = %q, want %q", spans.Spans[2].ParentID, spans.Spans[1].SpanID)
	}
	if spans.Spans[2].Memory == nil || spans.Spans[2].Memory.TotalAllocBytesDelta == 0 || spans.Spans[2].Memory.MallocsDelta == 0 {
		t.Fatalf("grandchild memory = %#v, want nonzero allocation sample", spans.Spans[2].Memory)
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

func TestSpanDocumentNodeComputesSelfDurationAndMemory(t *testing.T) {
	base := time.Date(2026, 6, 29, 8, 0, 0, 0, time.UTC)
	node := requestSpanDocumentNode{
		document: requestSpanDocument{
			StartedAt: base,
			EndedAt:   base.Add(10 * time.Millisecond),
			Memory: &requestSpanMemoryDocument{
				TotalAllocBytesDelta: 1000,
				MallocsDelta:         10,
				FreesDelta:           4,
			},
		},
		children: []requestSpanDocumentNode{
			{
				document: requestSpanDocument{
					StartedAt: base.Add(1 * time.Millisecond),
					EndedAt:   base.Add(4 * time.Millisecond),
					Memory: &requestSpanMemoryDocument{
						TotalAllocBytesDelta: 200,
						MallocsDelta:         3,
						FreesDelta:           1,
					},
				},
			},
			{
				document: requestSpanDocument{
					StartedAt: base.Add(3 * time.Millisecond),
					EndedAt:   base.Add(7 * time.Millisecond),
					Memory: &requestSpanMemoryDocument{
						TotalAllocBytesDelta: 300,
						MallocsDelta:         4,
						FreesDelta:           1,
					},
				},
			},
		},
	}

	node.document.SelfDurationMS = durationMilliseconds(selfSpanDuration(node.document, node.children))
	node.document.Memory.SelfTotalAllocBytesDelta = selfUint64(node.document.Memory.TotalAllocBytesDelta, node.children, func(memory requestSpanMemoryDocument) uint64 {
		return memory.TotalAllocBytesDelta
	})
	node.document.Memory.SelfMallocsDelta = selfUint64(node.document.Memory.MallocsDelta, node.children, func(memory requestSpanMemoryDocument) uint64 {
		return memory.MallocsDelta
	})
	node.document.Memory.SelfFreesDelta = selfUint64(node.document.Memory.FreesDelta, node.children, func(memory requestSpanMemoryDocument) uint64 {
		return memory.FreesDelta
	})

	if math.Abs(node.document.SelfDurationMS-4) > 0.001 {
		t.Fatalf("self duration = %fms, want 4ms", node.document.SelfDurationMS)
	}
	if node.document.Memory.SelfTotalAllocBytesDelta != 500 {
		t.Fatalf("self total alloc = %d, want 500", node.document.Memory.SelfTotalAllocBytesDelta)
	}
	if node.document.Memory.SelfMallocsDelta != 3 {
		t.Fatalf("self mallocs = %d, want 3", node.document.Memory.SelfMallocsDelta)
	}
	if node.document.Memory.SelfFreesDelta != 2 {
		t.Fatalf("self frees = %d, want 2", node.document.Memory.SelfFreesDelta)
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

func spanEventsContainAttr(events []struct {
	Name       string                 `json:"name"`
	Attributes map[string]interface{} `json:"attributes"`
}, name string, value interface{}) bool {
	for _, event := range events {
		if event.Name == "log" && event.Attributes[name] == value {
			return true
		}
	}
	return false
}

func allocateForCaptureTest(size int) []byte {
	data := make([]byte, size)
	for index := range data {
		data[index] = byte(index)
	}
	return data
}
