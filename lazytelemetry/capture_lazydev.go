//go:build lazydev

package lazytelemetry

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	runtimetrace "runtime/trace"
	"sort"
	"strings"
	"sync"
	"time"

	"golazy.dev/lazytelemetry/lazytracing"
)

var runtimeTraceMu sync.Mutex

type requestCapture struct {
	requestID       string
	fileID          string
	dir             string
	traceFile       *os.File
	traceStarted    bool
	locked          bool
	startedAt       time.Time
	startMem        runtime.MemStats
	startGoroutines int
	errors          []string
}

type requestCaptureDocument struct {
	RequestID  string                 `json:"request_id"`
	Method     string                 `json:"method"`
	Path       string                 `json:"path"`
	Status     int                    `json:"status"`
	Bytes      int                    `json:"bytes"`
	StartedAt  time.Time              `json:"started_at"`
	EndedAt    time.Time              `json:"ended_at"`
	DurationMS float64                `json:"duration_ms"`
	Panicked   bool                   `json:"panicked,omitempty"`
	Panic      string                 `json:"panic,omitempty"`
	TraceFile  string                 `json:"trace_file"`
	SpansFile  string                 `json:"spans_file"`
	LogsFile   string                 `json:"logs_file"`
	HandledBy  string                 `json:"handled_by,omitempty"`
	Category   string                 `json:"category,omitempty"`
	Runtime    requestRuntimeSummary  `json:"runtime"`
	Memory     requestMemorySummary   `json:"memory"`
	Spans      []requestSpanDocument  `json:"spans"`
	Errors     []string               `json:"errors,omitempty"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

type requestRuntimeSummary struct {
	GoVersion       string `json:"go_version"`
	GOOS            string `json:"goos"`
	GOARCH          string `json:"goarch"`
	NumCPU          int    `json:"num_cpu"`
	GoroutinesStart int    `json:"goroutines_start"`
	GoroutinesEnd   int    `json:"goroutines_end"`
}

type requestMemorySummary struct {
	AllocBytesStart      uint64  `json:"alloc_bytes_start"`
	AllocBytesEnd        uint64  `json:"alloc_bytes_end"`
	HeapAllocBytesStart  uint64  `json:"heap_alloc_bytes_start"`
	HeapAllocBytesEnd    uint64  `json:"heap_alloc_bytes_end"`
	HeapObjectsStart     uint64  `json:"heap_objects_start"`
	HeapObjectsEnd       uint64  `json:"heap_objects_end"`
	HeapObjectsDelta     int64   `json:"heap_objects_delta"`
	TotalAllocBytesDelta uint64  `json:"total_alloc_bytes_delta"`
	MallocsDelta         uint64  `json:"mallocs_delta"`
	FreesDelta           uint64  `json:"frees_delta"`
	SysBytesStart        uint64  `json:"sys_bytes_start"`
	SysBytesEnd          uint64  `json:"sys_bytes_end"`
	StackInuseBytesStart uint64  `json:"stack_inuse_bytes_start"`
	StackInuseBytesEnd   uint64  `json:"stack_inuse_bytes_end"`
	NextGCBytesStart     uint64  `json:"next_gc_bytes_start"`
	NextGCBytesEnd       uint64  `json:"next_gc_bytes_end"`
	NumGCStart           uint32  `json:"num_gc_start"`
	NumGCEnd             uint32  `json:"num_gc_end"`
	NumGCDelta           uint32  `json:"num_gc_delta"`
	PauseTotalNsStart    uint64  `json:"pause_total_ns_start"`
	PauseTotalNsEnd      uint64  `json:"pause_total_ns_end"`
	PauseTotalNsDelta    uint64  `json:"pause_total_ns_delta"`
	LastGCUnixNanoStart  uint64  `json:"last_gc_unix_nano_start"`
	LastGCUnixNanoEnd    uint64  `json:"last_gc_unix_nano_end"`
	GCCPUFractionStart   float64 `json:"gc_cpu_fraction_start"`
	GCCPUFractionEnd     float64 `json:"gc_cpu_fraction_end"`
}

type requestSpanDocument struct {
	Name           string                     `json:"name"`
	TraceID        string                     `json:"trace_id"`
	SpanID         string                     `json:"span_id"`
	ParentID       string                     `json:"parent_id,omitempty"`
	GoroutineID    uint64                     `json:"goroutine_id,omitempty"`
	StartedAt      time.Time                  `json:"started_at"`
	EndedAt        time.Time                  `json:"ended_at"`
	DurationMS     float64                    `json:"duration_ms"`
	SelfDurationMS float64                    `json:"self_duration_ms"`
	Memory         *requestSpanMemoryDocument `json:"memory,omitempty"`
	Attributes     map[string]interface{}     `json:"attributes,omitempty"`
	Events         []requestSpanEvent         `json:"events,omitempty"`
	Error          string                     `json:"error,omitempty"`
}

type requestSpanMemoryDocument struct {
	TotalAllocBytesDelta     uint64 `json:"total_alloc_bytes_delta"`
	MallocsDelta             uint64 `json:"mallocs_delta"`
	FreesDelta               uint64 `json:"frees_delta"`
	SelfTotalAllocBytesDelta uint64 `json:"self_total_alloc_bytes_delta"`
	SelfMallocsDelta         uint64 `json:"self_mallocs_delta"`
	SelfFreesDelta           uint64 `json:"self_frees_delta"`
}

type requestSpanEvent struct {
	Name       string                 `json:"name"`
	Time       time.Time              `json:"time"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

func beginRequestCapture(enabled bool, requestID string) *requestCapture {
	if !enabled {
		return nil
	}
	runtimeTraceMu.Lock()
	capture := &requestCapture{
		requestID: strings.TrimSpace(requestID),
		fileID:    traceFileID(requestID),
		dir:       requestCaptureDirectory,
		startedAt: time.Now(),
		locked:    true,
	}
	runtime.ReadMemStats(&capture.startMem)
	capture.startGoroutines = runtime.NumGoroutine()

	if err := os.MkdirAll(capture.dir, 0o755); err != nil {
		capture.errors = append(capture.errors, fmt.Sprintf("create trace directory: %v", err))
		return capture
	}
	traceFile, err := os.Create(capture.path(".trace"))
	if err != nil {
		capture.errors = append(capture.errors, fmt.Sprintf("create runtime trace: %v", err))
		return capture
	}
	if err := runtimetrace.Start(traceFile); err != nil {
		capture.errors = append(capture.errors, fmt.Sprintf("start runtime trace: %v", err))
		_ = traceFile.Close()
		return capture
	}
	capture.traceFile = traceFile
	capture.traceStarted = true
	return capture
}

func (capture *requestCapture) Finish(result requestCaptureResult, span *lazytracing.Span) {
	if capture == nil {
		return
	}
	var endMem runtime.MemStats
	runtime.ReadMemStats(&endMem)
	endGoroutines := runtime.NumGoroutine()

	if capture.traceStarted {
		runtimetrace.Stop()
	}
	if capture.traceFile != nil {
		if err := capture.traceFile.Close(); err != nil {
			capture.errors = append(capture.errors, fmt.Sprintf("close runtime trace: %v", err))
		}
	}
	if capture.locked {
		capture.locked = false
		runtimeTraceMu.Unlock()
	}

	document := capture.document(result, span, endMem, endGoroutines)
	if err := capture.writeSpans(document); err != nil {
		capture.errors = append(capture.errors, fmt.Sprintf("write spans: %v", err))
	}
	if err := capture.writeLogs(result, span); err != nil {
		capture.errors = append(capture.errors, fmt.Sprintf("write logs: %v", err))
	}
	lazytracing.ClearAllocationSamples(span)
}

func (capture *requestCapture) document(result requestCaptureResult, span *lazytracing.Span, endMem runtime.MemStats, endGoroutines int) requestCaptureDocument {
	if result.StartedAt.IsZero() {
		result.StartedAt = capture.startedAt
	}
	if result.EndedAt.IsZero() {
		result.EndedAt = time.Now()
	}
	if result.Duration == 0 {
		result.Duration = result.EndedAt.Sub(result.StartedAt)
	}
	var panicValue string
	if result.Panic != nil {
		panicValue = fmt.Sprint(result.Panic)
	}
	spans := spanDocuments(span)
	handledBy, category := requestHandledBy(spans)
	return requestCaptureDocument{
		RequestID:  result.RequestID,
		Method:     result.Method,
		Path:       result.Path,
		Status:     result.Status,
		Bytes:      result.Bytes,
		StartedAt:  result.StartedAt,
		EndedAt:    result.EndedAt,
		DurationMS: durationMilliseconds(result.Duration),
		Panicked:   result.Panic != nil,
		Panic:      panicValue,
		TraceFile:  filepath.ToSlash(capture.path(".trace")),
		SpansFile:  filepath.ToSlash(capture.path(".spans")),
		LogsFile:   filepath.ToSlash(capture.path(".log.json")),
		HandledBy:  handledBy,
		Category:   category,
		Runtime: requestRuntimeSummary{
			GoVersion:       runtime.Version(),
			GOOS:            runtime.GOOS,
			GOARCH:          runtime.GOARCH,
			NumCPU:          runtime.NumCPU(),
			GoroutinesStart: capture.startGoroutines,
			GoroutinesEnd:   endGoroutines,
		},
		Memory: memorySummary(capture.startMem, endMem),
		Spans:  spans,
		Errors: append([]string(nil), capture.errors...),
	}
}

func (capture *requestCapture) writeSpans(document requestCaptureDocument) error {
	file, err := os.Create(capture.path(".spans"))
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(document)
}

func (capture *requestCapture) writeLogs(result requestCaptureResult, span *lazytracing.Span) error {
	file, err := os.Create(capture.path(".log.json"))
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	if span == nil {
		return nil
	}
	for _, eventSpan := range spansInOrder(span) {
		for _, event := range eventSpan.Events() {
			if event.Name != "log" {
				continue
			}
			record := map[string]interface{}{
				"time":       event.Time.Format(time.RFC3339Nano),
				"request_id": result.RequestID,
				"trace_id":   eventSpan.TraceID(),
				"span_id":    eventSpan.SpanID(),
				"method":     result.Method,
				"path":       result.Path,
			}
			for key, value := range attrsMap(event.Attributes) {
				record[key] = value
			}
			if err := encoder.Encode(record); err != nil {
				return err
			}
		}
	}
	return nil
}

func (capture *requestCapture) path(extension string) string {
	return filepath.Join(capture.dir, capture.fileID+extension)
}

func traceFileID(requestID string) string {
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return "request"
	}
	var builder strings.Builder
	for _, char := range requestID {
		if char >= 'a' && char <= 'z' ||
			char >= 'A' && char <= 'Z' ||
			char >= '0' && char <= '9' ||
			char == '_' || char == '-' || char == '.' {
			builder.WriteRune(char)
			continue
		}
		builder.WriteByte('_')
	}
	if builder.Len() == 0 {
		return "request"
	}
	return builder.String()
}

func spansInOrder(span *lazytracing.Span) []*lazytracing.Span {
	if span == nil {
		return nil
	}
	spans := []*lazytracing.Span{span}
	for _, child := range span.Children() {
		spans = append(spans, spansInOrder(child)...)
	}
	return spans
}

func spanDocuments(span *lazytracing.Span) []requestSpanDocument {
	if span == nil {
		return nil
	}
	root := spanDocumentNode(span)
	documents := make([]requestSpanDocument, 0, len(spansInOrder(span)))
	appendSpanDocuments(&documents, root)
	return documents
}

type requestSpanDocumentNode struct {
	document requestSpanDocument
	children []requestSpanDocumentNode
}

func spanDocumentNode(span *lazytracing.Span) requestSpanDocumentNode {
	node := requestSpanDocumentNode{document: spanDocument(span)}
	for _, child := range span.Children() {
		node.children = append(node.children, spanDocumentNode(child))
	}
	node.document.SelfDurationMS = durationMilliseconds(selfSpanDuration(node.document, node.children))
	if node.document.Memory != nil {
		node.document.Memory.SelfTotalAllocBytesDelta = selfUint64(node.document.Memory.TotalAllocBytesDelta, node.children, func(memory requestSpanMemoryDocument) uint64 {
			return memory.TotalAllocBytesDelta
		})
		node.document.Memory.SelfMallocsDelta = selfUint64(node.document.Memory.MallocsDelta, node.children, func(memory requestSpanMemoryDocument) uint64 {
			return memory.MallocsDelta
		})
		node.document.Memory.SelfFreesDelta = selfUint64(node.document.Memory.FreesDelta, node.children, func(memory requestSpanMemoryDocument) uint64 {
			return memory.FreesDelta
		})
	}
	return node
}

func appendSpanDocuments(documents *[]requestSpanDocument, node requestSpanDocumentNode) {
	*documents = append(*documents, node.document)
	for _, child := range node.children {
		appendSpanDocuments(documents, child)
	}
}

func spanDocument(span *lazytracing.Span) requestSpanDocument {
	if span == nil {
		return requestSpanDocument{}
	}
	var errorMessage string
	if err := span.Error(); err != nil {
		errorMessage = err.Error()
	}
	events := span.Events()
	eventDocuments := make([]requestSpanEvent, 0, len(events))
	for _, event := range events {
		eventDocuments = append(eventDocuments, requestSpanEvent{
			Name:       event.Name,
			Time:       event.Time,
			Attributes: attrsMap(event.Attributes),
		})
	}
	document := requestSpanDocument{
		Name:        span.Name(),
		TraceID:     span.TraceID(),
		SpanID:      span.SpanID(),
		ParentID:    span.ParentID(),
		GoroutineID: span.GoroutineID(),
		StartedAt:   span.StartedAt(),
		EndedAt:     span.EndedAt(),
		DurationMS:  durationMilliseconds(span.Duration()),
		Attributes:  attrsMap(span.Attributes()),
		Events:      eventDocuments,
		Error:       errorMessage,
	}
	if memory, ok := lazytracing.SpanAllocationSummary(span); ok {
		document.Memory = &requestSpanMemoryDocument{
			TotalAllocBytesDelta: memory.TotalAllocBytesDelta,
			MallocsDelta:         memory.MallocsDelta,
			FreesDelta:           memory.FreesDelta,
		}
	}
	return document
}

func requestHandledBy(spans []requestSpanDocument) (string, string) {
	var handledBy string
	for _, span := range spans {
		if !spanHandledRequest(span) {
			continue
		}
		if name, ok := span.Attributes["middleware.name"].(string); ok && strings.TrimSpace(name) != "" {
			handledBy = strings.TrimSpace(name)
			continue
		}
		handledBy = strings.TrimPrefix(span.Name, "middleware ")
	}
	if handledBy == "" {
		return "", "other"
	}
	switch handledBy {
	case "lazydispatch.Router", "lazydispatch.RouteOnly":
		return handledBy, "framework"
	case "lazyassets.Registry":
		return handledBy, "assets"
	default:
		return handledBy, "other"
	}
}

func spanHandledRequest(span requestSpanDocument) bool {
	if span.Attributes == nil {
		return false
	}
	if handled, ok := span.Attributes["middleware.handled"].(bool); ok {
		return handled
	}
	if nextCalled, ok := span.Attributes["middleware.next_called"].(bool); ok {
		return !nextCalled
	}
	return false
}

func selfSpanDuration(span requestSpanDocument, children []requestSpanDocumentNode) time.Duration {
	duration := span.EndedAt.Sub(span.StartedAt)
	if duration <= 0 {
		return 0
	}
	covered := childCoveredDuration(span, children)
	if covered >= duration {
		return 0
	}
	return duration - covered
}

func childCoveredDuration(span requestSpanDocument, children []requestSpanDocumentNode) time.Duration {
	type interval struct {
		start time.Time
		end   time.Time
	}
	intervals := make([]interval, 0, len(children))
	for _, child := range children {
		start := child.document.StartedAt
		end := child.document.EndedAt
		if start.Before(span.StartedAt) {
			start = span.StartedAt
		}
		if end.After(span.EndedAt) {
			end = span.EndedAt
		}
		if end.After(start) {
			intervals = append(intervals, interval{start: start, end: end})
		}
	}
	if len(intervals) == 0 {
		return 0
	}
	sort.Slice(intervals, func(i, j int) bool {
		return intervals[i].start.Before(intervals[j].start)
	})
	current := intervals[0]
	var covered time.Duration
	for _, next := range intervals[1:] {
		if !next.start.After(current.end) {
			if next.end.After(current.end) {
				current.end = next.end
			}
			continue
		}
		covered += current.end.Sub(current.start)
		current = next
	}
	covered += current.end.Sub(current.start)
	return covered
}

func selfUint64(total uint64, children []requestSpanDocumentNode, value func(requestSpanMemoryDocument) uint64) uint64 {
	var childTotal uint64
	for _, child := range children {
		if child.document.Memory == nil {
			continue
		}
		childTotal += value(*child.document.Memory)
	}
	if childTotal >= total {
		return 0
	}
	return total - childTotal
}

func attrsMap(attrs []slog.Attr) map[string]interface{} {
	if len(attrs) == 0 {
		return nil
	}
	result := make(map[string]interface{}, len(attrs))
	for _, attr := range attrs {
		if attr.Key == "" {
			continue
		}
		result[attr.Key] = attrValue(attr.Value)
	}
	return result
}

func attrValue(value slog.Value) interface{} {
	value = value.Resolve()
	switch value.Kind() {
	case slog.KindString:
		return value.String()
	case slog.KindBool:
		return value.Bool()
	case slog.KindInt64:
		return value.Int64()
	case slog.KindUint64:
		return value.Uint64()
	case slog.KindFloat64:
		return value.Float64()
	case slog.KindDuration:
		return value.Duration().String()
	case slog.KindTime:
		return value.Time().Format(time.RFC3339Nano)
	case slog.KindGroup:
		return attrsMap(value.Group())
	case slog.KindAny:
		return safeAny(value.Any())
	default:
		return value.String()
	}
}

func safeAny(value interface{}) interface{} {
	switch value := value.(type) {
	case nil, string, bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return value
	case fmt.Stringer:
		return value.String()
	case error:
		return value.Error()
	default:
		return fmt.Sprint(value)
	}
}

func memorySummary(start, end runtime.MemStats) requestMemorySummary {
	return requestMemorySummary{
		AllocBytesStart:      start.Alloc,
		AllocBytesEnd:        end.Alloc,
		HeapAllocBytesStart:  start.HeapAlloc,
		HeapAllocBytesEnd:    end.HeapAlloc,
		HeapObjectsStart:     start.HeapObjects,
		HeapObjectsEnd:       end.HeapObjects,
		HeapObjectsDelta:     int64(end.HeapObjects) - int64(start.HeapObjects),
		TotalAllocBytesDelta: uint64Delta(end.TotalAlloc, start.TotalAlloc),
		MallocsDelta:         uint64Delta(end.Mallocs, start.Mallocs),
		FreesDelta:           uint64Delta(end.Frees, start.Frees),
		SysBytesStart:        start.Sys,
		SysBytesEnd:          end.Sys,
		StackInuseBytesStart: start.StackInuse,
		StackInuseBytesEnd:   end.StackInuse,
		NextGCBytesStart:     start.NextGC,
		NextGCBytesEnd:       end.NextGC,
		NumGCStart:           start.NumGC,
		NumGCEnd:             end.NumGC,
		NumGCDelta:           uint32Delta(end.NumGC, start.NumGC),
		PauseTotalNsStart:    start.PauseTotalNs,
		PauseTotalNsEnd:      end.PauseTotalNs,
		PauseTotalNsDelta:    uint64Delta(end.PauseTotalNs, start.PauseTotalNs),
		LastGCUnixNanoStart:  start.LastGC,
		LastGCUnixNanoEnd:    end.LastGC,
		GCCPUFractionStart:   start.GCCPUFraction,
		GCCPUFractionEnd:     end.GCCPUFraction,
	}
}

func durationMilliseconds(duration time.Duration) float64 {
	return float64(duration) / float64(time.Millisecond)
}

func uint64Delta(end, start uint64) uint64 {
	if end < start {
		return 0
	}
	return end - start
}

func uint32Delta(end, start uint32) uint32 {
	if end < start {
		return 0
	}
	return end - start
}
