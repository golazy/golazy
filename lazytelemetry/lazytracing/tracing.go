// Package lazytracing provides lightweight span helpers for GoLazy telemetry.
package lazytracing

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"runtime/trace"
	"strings"
	"sync"
	"time"
)

type contextKey struct{}
type traceContextKey struct{}

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
	startedAt  time.Time
	endedAt    time.Time
	attributes []slog.Attr
	events     []Event
	err        error
	ended      bool

	task   *trace.Task
	region *trace.Region
}

// WithTraceContext attaches trace context to ctx.
func WithTraceContext(ctx context.Context, traceContext TraceContext) context.Context {
	if traceContext.TraceID == "" {
		return ctx
	}
	return context.WithValue(ctx, traceContextKey{}, traceContext)
}

// TraceContextFromContext returns the trace context attached to ctx.
func TraceContextFromContext(ctx context.Context) (TraceContext, bool) {
	traceContext, ok := ctx.Value(traceContextKey{}).(TraceContext)
	return traceContext, ok
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
	traceContext, ok := TraceContextFromContext(ctx)
	if !ok || traceContext.TraceID == "" {
		traceContext.TraceID = randomHex(16)
	}
	parentID := traceContext.SpanID
	traceContext.SpanID = randomHex(8)
	traceContext.Remote = false
	ctx = WithTraceContext(ctx, traceContext)

	var task *trace.Task
	if parentID == "" {
		ctx, task = trace.NewTask(ctx, name)
	}
	region := trace.StartRegion(ctx, name)

	span := &Span{
		name:       name,
		traceID:    traceContext.TraceID,
		spanID:     traceContext.SpanID,
		parentID:   parentID,
		startedAt:  time.Now(),
		attributes: append([]slog.Attr(nil), attributes...),
		task:       task,
		region:     region,
	}
	return context.WithValue(ctx, contextKey{}, span), span
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

// AddAttributes appends attributes to the span.
func (s *Span) AddAttributes(attributes ...slog.Attr) {
	if s == nil || len(attributes) == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.attributes = append(s.attributes, attributes...)
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

// RecordError records err on the span.
func (s *Span) RecordError(err error) {
	if s == nil || err == nil {
		return
	}
	s.mu.Lock()
	s.err = err
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
	region := s.region
	task := s.task
	s.mu.Unlock()

	if region != nil {
		region.End()
	}
	if task != nil {
		task.End()
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
