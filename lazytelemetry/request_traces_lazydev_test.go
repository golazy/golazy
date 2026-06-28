//go:build lazydev

package lazytelemetry

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golazy.dev/lazycontrolplane"
)

func TestLazyDevRequestTracesHandlerListsCapturesWithLogs(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.MkdirAll(requestCaptureDirectory, 0o755); err != nil {
		t.Fatal(err)
	}

	startedAt := time.Date(2026, 6, 28, 23, 41, 38, 0, time.UTC)
	document := requestCaptureDocument{
		RequestID:  "req-123",
		Method:     http.MethodGet,
		Path:       "/pools",
		Status:     http.StatusOK,
		StartedAt:  startedAt,
		EndedAt:    startedAt.Add(12 * time.Millisecond),
		DurationMS: 12,
		TraceFile:  ".tmp/traces/req-123.trace",
		SpansFile:  ".tmp/traces/req-123.spans",
		LogsFile:   ".tmp/traces/req-123.log.json",
		Spans: []requestSpanDocument{
			{Name: "http.server.request", SpanID: "root", StartedAt: startedAt, EndedAt: startedAt.Add(12 * time.Millisecond), DurationMS: 12},
			{Name: "controller.action accounts.Pools", SpanID: "child", ParentID: "root", StartedAt: startedAt.Add(time.Millisecond), EndedAt: startedAt.Add(4 * time.Millisecond), DurationMS: 3},
		},
	}
	writeJSONFile(t, filepath.Join(requestCaptureDirectory, "req-123.spans"), document)
	if err := os.WriteFile(filepath.Join(requestCaptureDirectory, "req-123.log.json"), []byte(`{"message":"request completed","level":"info","request_id":"req-123"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	plane := lazycontrolplane.New(lazycontrolplane.Config{})
	RegisterLazyDevHandlers(plane)

	response := httptest.NewRecorder()
	plane.ServeHTTP(response, httptest.NewRequest(http.MethodGet, LazyDevRequestTracesPath, nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if got := response.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
	var got struct {
		Directory string `json:"directory"`
		Traces    []struct {
			RequestID string `json:"request_id"`
			Method    string `json:"method"`
			Path      string `json:"path"`
			Spans     []struct {
				Name string `json:"name"`
			} `json:"spans"`
			Logs []map[string]interface{} `json:"logs"`
		} `json:"traces"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v\n%s", err, response.Body.String())
	}
	if got.Directory != ".tmp/traces" {
		t.Fatalf("directory = %q, want .tmp/traces", got.Directory)
	}
	if len(got.Traces) != 1 {
		t.Fatalf("traces = %#v, want one trace", got.Traces)
	}
	trace := got.Traces[0]
	if trace.RequestID != "req-123" || trace.Method != http.MethodGet || trace.Path != "/pools" {
		t.Fatalf("trace summary = %#v, want req-123 GET /pools", trace)
	}
	if len(trace.Spans) != 2 || trace.Spans[1].Name != "controller.action accounts.Pools" {
		t.Fatalf("spans = %#v, want controller action", trace.Spans)
	}
	if len(trace.Logs) != 1 || trace.Logs[0]["message"] != "request completed" {
		t.Fatalf("logs = %#v, want request completed log", trace.Logs)
	}
}

func writeJSONFile(t *testing.T, path string, value interface{}) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := json.NewEncoder(file).Encode(value); err != nil {
		t.Fatal(err)
	}
}
