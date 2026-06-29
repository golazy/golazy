// Package lazytracing provides lightweight span helpers for GoLazy telemetry.
package lazytracing

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	runtimetrace "runtime/trace"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type contextKey struct{}
type traceContextKey struct{}
type requestIDKey struct{}

// TraceContext stores W3C trace context identifiers without depending on an
// OpenTelemetry SDK.
type TraceContext struct {
	TraceID    string
	SpanID     string
	TraceFlags string
	TraceState string
	Remote     bool
}

// Event records a named span event.
type Event struct {
	Name       string
	Time       time.Time
	Attributes []slog.Attr
}

// Span is a lightweight in-process span.
type Span struct {
	mu sync.Mutex

	name       string
	traceID    string
	spanID     string
	parentID   string
	goroutine  uint64
	startedAt  time.Time
	endedAt    time.Time
	attributes []slog.Attr
	events     []Event
	children   []*Span
	err        error
	ended      bool

	parent   *Span
	otelSpan oteltrace.Span
	task     *runtimetrace.Task
	region   *runtimetrace.Region
}

// WithTraceContext attaches trace context to ctx.
func WithTraceContext(ctx context.Context, traceContext TraceContext) context.Context {
	if traceContext.TraceID == "" {
		return ctx
	}
	if traceContext.Remote {
		if spanContext := otelSpanContext(traceContext); spanContext.IsValid() {
			ctx = oteltrace.ContextWithRemoteSpanContext(ctx, spanContext)
		}
	}
	return context.WithValue(ctx, traceContextKey{}, traceContext)
}

// TraceContextFromContext returns the trace context attached to ctx.
func TraceContextFromContext(ctx context.Context) (TraceContext, bool) {
	traceContext, ok := ctx.Value(traceContextKey{}).(TraceContext)
	return traceContext, ok
}

// WithRequestID attaches requestID to ctx for span and runtime trace
// correlation.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return ctx
	}
	return context.WithValue(ctx, requestIDKey{}, requestID)
}

// RequestID returns the request id attached to ctx.
func RequestID(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDKey{}).(string)
	return requestID
}

// ParseTraceparent parses the W3C traceparent header.
func ParseTraceparent(traceparent, tracestate string) (TraceContext, bool) {
	parts := strings.Split(strings.TrimSpace(traceparent), "-")
	if len(parts) != 4 {
		return TraceContext{}, false
	}
	version, traceID, spanID, flags := parts[0], strings.ToLower(parts[1]), strings.ToLower(parts[2]), strings.ToLower(parts[3])
	if len(version) != 2 || len(traceID) != 32 || len(spanID) != 16 || len(flags) != 2 {
		return TraceContext{}, false
	}
	if !isLowerHex(version) || !isLowerHex(traceID) || !isLowerHex(spanID) || !isLowerHex(flags) {
		return TraceContext{}, false
	}
	if version == "ff" || allZero(traceID) || allZero(spanID) {
		return TraceContext{}, false
	}
	return TraceContext{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: flags,
		TraceState: strings.TrimSpace(tracestate),
		Remote:     true,
	}, true
}

// StartSpan starts a lightweight span and returns a context containing it.
func StartSpan(ctx context.Context, name string, attributes ...slog.Attr) (context.Context, *Span) {
	if name == "" {
		name = "span"
	}
	attributes = spanAttributesFromContext(ctx, attributes)
	parentSpan := SpanFromContext(ctx)
	traceContext, ok := TraceContextFromContext(ctx)
	if !ok || traceContext.TraceID == "" {
		traceContext.TraceID = randomHex(16)
	}
	parentID := traceContext.SpanID
	if parentSpan != nil {
		parentID = parentSpan.SpanID()
	} else if spanContext := oteltrace.SpanContextFromContext(ctx); spanContext.IsValid() {
		traceContext.TraceID = spanContext.TraceID().String()
		parentID = spanContext.SpanID().String()
	}
	ctx, otelSpan := otel.Tracer("golazy.dev/lazytelemetry").Start(ctx, name,
		oteltrace.WithAttributes(slogAttrsToOTel(attributes)...),
	)
	spanContext := otelSpan.SpanContext()
	if spanContext.IsValid() && spanContext.SpanID().String() != parentID {
		traceContext.TraceID = spanContext.TraceID().String()
		traceContext.SpanID = spanContext.SpanID().String()
	} else {
		traceContext.SpanID = randomHex(8)
	}
	traceContext.Remote = false
	ctx = context.WithValue(ctx, traceContextKey{}, traceContext)

	var task *runtimetrace.Task
	if parentSpan == nil {
		ctx, task = runtimetrace.NewTask(ctx, name)
	}
	region := runtimetrace.StartRegion(ctx, name)
	if requestID := RequestID(ctx); requestID != "" {
		runtimetrace.Log(ctx, "request_id", requestID)
	}

	span := &Span{
		name:       name,
		traceID:    traceContext.TraceID,
		spanID:     traceContext.SpanID,
		parentID:   parentID,
		goroutine:  currentGoroutineID(),
		startedAt:  time.Now(),
		attributes: append([]slog.Attr(nil), attributes...),
		parent:     parentSpan,
		otelSpan:   otelSpan,
		task:       task,
		region:     region,
	}
	if parentSpan != nil {
		parentSpan.addChild(span)
	}
	startSpanAllocationSample(ctx, span)
	return context.WithValue(ctx, contextKey{}, span), span
}

func spanAttributesFromContext(ctx context.Context, attributes []slog.Attr) []slog.Attr {
	requestID := RequestID(ctx)
	if requestID == "" || hasAttr(attributes, "request_id") {
		return attributes
	}
	attributes = append(append([]slog.Attr(nil), attributes...), slog.String("request_id", requestID))
	return attributes
}

// StartRegion starts a child span and Go runtime trace region when ctx already
// carries an active span. It returns ctx unchanged and a nil span when telemetry
// is not active for the request.
func StartRegion(ctx context.Context, name string, attributes ...slog.Attr) (context.Context, *Span) {
	if SpanFromContext(ctx) == nil {
		return ctx, nil
	}
	return StartSpan(ctx, name, attributes...)
}

// SpanFromContext returns the active span attached to ctx.
func SpanFromContext(ctx context.Context) *Span {
	span, _ := ctx.Value(contextKey{}).(*Span)
	return span
}

// TraceID returns the active trace id from ctx.
func TraceID(ctx context.Context) string {
	if traceContext, ok := TraceContextFromContext(ctx); ok {
		return traceContext.TraceID
	}
	return ""
}

// SpanID returns the active span id from ctx.
func SpanID(ctx context.Context) string {
	if traceContext, ok := TraceContextFromContext(ctx); ok {
		return traceContext.SpanID
	}
	return ""
}

// Log records a message in the Go runtime trace when ctx carries an active
// span. Span event recording is owned by the logging package.
func Log(ctx context.Context, category string, message string) {
	if SpanFromContext(ctx) == nil {
		return
	}
	category = strings.TrimSpace(category)
	if category == "" {
		category = "event"
	}
	runtimetrace.Log(ctx, category, message)
}

// Name returns the span name.
func (s *Span) Name() string {
	if s == nil {
		return ""
	}
	return s.name
}

// TraceID returns the span trace id.
func (s *Span) TraceID() string {
	if s == nil {
		return ""
	}
	return s.traceID
}

// SpanID returns the span id.
func (s *Span) SpanID() string {
	if s == nil {
		return ""
	}
	return s.spanID
}

// ParentID returns the parent span id when known.
func (s *Span) ParentID() string {
	if s == nil {
		return ""
	}
	return s.parentID
}

// GoroutineID returns the goroutine identifier captured for development
// diagnostics. Production builds return zero.
func (s *Span) GoroutineID() uint64 {
	if s == nil {
		return 0
	}
	return s.goroutine
}

// StartedAt returns the time the span started.
func (s *Span) StartedAt() time.Time {
	if s == nil {
		return time.Time{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.startedAt
}

// EndedAt returns the time the span ended, or the zero time when it is still open.
func (s *Span) EndedAt() time.Time {
	if s == nil {
		return time.Time{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.endedAt
}

// AddAttributes appends attributes to the span.
func (s *Span) AddAttributes(attributes ...slog.Attr) {
	if s == nil || len(attributes) == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.attributes = append(s.attributes, attributes...)
	if s.otelSpan != nil {
		s.otelSpan.SetAttributes(slogAttrsToOTel(attributes)...)
	}
}

// Attributes returns a copy of the span attributes.
func (s *Span) Attributes() []slog.Attr {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]slog.Attr(nil), s.attributes...)
}

// AddEvent records a span event.
func (s *Span) AddEvent(name string, attributes ...slog.Attr) {
	if s == nil || name == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, Event{
		Name:       name,
		Time:       time.Now(),
		Attributes: append([]slog.Attr(nil), attributes...),
	})
	if s.otelSpan != nil {
		s.otelSpan.AddEvent(name, oteltrace.WithAttributes(slogAttrsToOTel(attributes)...))
	}
}

// Events returns a copy of recorded span events.
func (s *Span) Events() []Event {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Event(nil), s.events...)
}

// Children returns a copy of direct child spans.
func (s *Span) Children() []*Span {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]*Span(nil), s.children...)
}

func (s *Span) addChild(child *Span) {
	if s == nil || child == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.children = append(s.children, child)
}

func hasAttr(attrs []slog.Attr, name string) bool {
	for _, attr := range attrs {
		if attr.Key == name {
			return true
		}
	}
	return false
}

// RecordError records err on the span.
func (s *Span) RecordError(err error) {
	if s == nil || err == nil {
		return
	}
	s.mu.Lock()
	s.err = err
	if s.otelSpan != nil {
		s.otelSpan.RecordError(err)
		s.otelSpan.SetStatus(codes.Error, err.Error())
	}
	s.mu.Unlock()
	s.AddEvent("exception", slog.String("error", err.Error()))
}

// Error returns the recorded span error.
func (s *Span) Error() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.err
}

// End finishes the span.
func (s *Span) End() {
	if s == nil {
		return
	}
	s.mu.Lock()
	if s.ended {
		s.mu.Unlock()
		return
	}
	s.ended = true
	s.endedAt = time.Now()
	finishSpanAllocationSample(s)
	region := s.region
	task := s.task
	otelSpan := s.otelSpan
	s.mu.Unlock()

	if region != nil {
		region.End()
	}
	if task != nil {
		task.End()
	}
	if otelSpan != nil {
		otelSpan.End()
	}
}

// Duration returns the span duration when it has ended, or the elapsed time
// since start otherwise.
func (s *Span) Duration() time.Duration {
	if s == nil || s.startedAt.IsZero() {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.endedAt.IsZero() {
		return s.endedAt.Sub(s.startedAt)
	}
	return time.Since(s.startedAt)
}

func randomHex(bytes int) string {
	data := make([]byte, bytes)
	if _, err := rand.Read(data); err != nil {
		panic(fmt.Errorf("lazytracing: generate id: %w", err))
	}
	return hex.EncodeToString(data)
}

func isLowerHex(value string) bool {
	for _, char := range value {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return false
		}
	}
	return true
}

func allZero(value string) bool {
	for _, char := range value {
		if char != '0' {
			return false
		}
	}
	return true
}

func otelSpanContext(traceContext TraceContext) oteltrace.SpanContext {
	traceID, err := oteltrace.TraceIDFromHex(traceContext.TraceID)
	if err != nil {
		return oteltrace.SpanContext{}
	}
	spanID, err := oteltrace.SpanIDFromHex(traceContext.SpanID)
	if err != nil {
		return oteltrace.SpanContext{}
	}
	traceFlags := oteltrace.TraceFlags(0)
	if traceContext.TraceFlags == "01" {
		traceFlags = oteltrace.FlagsSampled
	}
	traceState, err := oteltrace.ParseTraceState(traceContext.TraceState)
	if err != nil {
		traceState = oteltrace.TraceState{}
	}
	return oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: traceFlags,
		TraceState: traceState,
		Remote:     traceContext.Remote,
	})
}

func slogAttrsToOTel(attrs []slog.Attr) []attribute.KeyValue {
	if len(attrs) == 0 {
		return nil
	}
	values := make([]attribute.KeyValue, 0, len(attrs))
	for _, attr := range attrs {
		if attr.Key == "" {
			continue
		}
		values = append(values, slogAttrToOTel(attr))
	}
	return values
}

func slogAttrToOTel(attr slog.Attr) attribute.KeyValue {
	value := attr.Value.Resolve()
	key := attribute.Key(attr.Key)
	switch value.Kind() {
	case slog.KindBool:
		return key.Bool(value.Bool())
	case slog.KindDuration:
		return key.String(value.Duration().String())
	case slog.KindFloat64:
		return key.Float64(value.Float64())
	case slog.KindInt64:
		return key.Int64(value.Int64())
	case slog.KindString:
		return key.String(value.String())
	case slog.KindTime:
		return key.String(value.Time().Format(time.RFC3339Nano))
	case slog.KindUint64:
		return key.Int64(int64(value.Uint64()))
	default:
		return key.String(value.String())
	}
}
