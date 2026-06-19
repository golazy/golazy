package lazysse

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	contentType = "text/event-stream"
	cachePolicy = "no-cache, no-transform"
)

// Event is one Server-Sent Event frame.
type Event struct {
	Event   string
	ID      string
	Data    []string
	Comment []string
	Retry   time.Duration
}

// Option configures a stream before it starts.
type Option func(*options)

type options struct {
	status int
	header http.Header
}

// Status sets the HTTP status used when the stream starts.
func Status(code int) Option {
	return func(opts *options) {
		opts.status = code
	}
}

// Header adds a response header before the stream starts.
func Header(name string, value string) Option {
	return func(opts *options) {
		if opts.header == nil {
			opts.header = http.Header{}
		}
		opts.header.Add(name, value)
	}
}

// SubscribeOptions configures a subscription source.
type SubscribeOptions struct {
	LastEventID string
}

// Source produces events for a stream.
type Source interface {
	Subscribe(context.Context, SubscribeOptions) (Subscription, error)
}

// Subscription is a live stream of events.
type Subscription interface {
	Events() <-chan Event
	Close() error
}

// Stream writes SSE frames to a response.
type Stream struct {
	writer http.ResponseWriter
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.Mutex
}

// Start starts an SSE response.
func Start(w http.ResponseWriter, r *http.Request, opts ...Option) (*Stream, error) {
	if w == nil {
		return nil, fmt.Errorf("lazysse: response writer is nil")
	}
	if r == nil {
		return nil, fmt.Errorf("lazysse: request is nil")
	}
	if !supportsFlush(w) {
		return nil, fmt.Errorf("lazysse: response writer does not support flushing")
	}

	cfg := options{
		status: http.StatusOK,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	if cfg.status == 0 {
		cfg.status = http.StatusOK
	}
	if cfg.status < 100 || cfg.status > 999 {
		return nil, fmt.Errorf("lazysse: invalid response status %d", cfg.status)
	}

	header := w.Header()
	header.Set("Content-Type", contentType)
	header.Set("Cache-Control", cachePolicy)
	for key, values := range cfg.header {
		for _, value := range values {
			header.Add(key, value)
		}
	}

	writer, err := startStream(w, cfg.status)
	if err != nil {
		return nil, err
	}
	if err := http.NewResponseController(writer).Flush(); err != nil {
		return nil, fmt.Errorf("lazysse: flush stream headers: %w", err)
	}

	ctx, cancel := context.WithCancel(withRequest(r.Context(), r))
	return &Stream{
		writer: writer,
		ctx:    ctx,
		cancel: cancel,
	}, nil
}

// Serve starts a stream, runs fn, and closes the stream when fn returns.
func Serve(w http.ResponseWriter, r *http.Request, fn func(*Stream) error) error {
	stream, err := Start(w, r)
	if err != nil {
		return err
	}
	defer stream.Close()
	if fn == nil {
		return nil
	}
	return fn(stream)
}

// Context returns the stream context.
func (s *Stream) Context() context.Context {
	if s == nil || s.ctx == nil {
		return context.Background()
	}
	return s.ctx
}

// Done is closed when the client disconnects or the stream is closed.
func (s *Stream) Done() <-chan struct{} {
	return s.Context().Done()
}

// Close stops stream helpers such as heartbeats.
func (s *Stream) Close() error {
	if s == nil || s.cancel == nil {
		return nil
	}
	s.cancel()
	return nil
}

// LastEventID returns the browser's Last-Event-ID request header.
func (s *Stream) LastEventID() (string, bool) {
	if s == nil {
		return "", false
	}
	request, ok := requestFromContext(s.Context())
	if !ok {
		return "", false
	}
	id := strings.TrimSpace(request.Header.Get("Last-Event-ID"))
	return id, id != ""
}

// Heartbeat writes SSE comments on interval until the stream is closed.
func (s *Stream) Heartbeat(interval time.Duration) {
	if s == nil || interval <= 0 {
		return
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-s.Done():
				return
			case <-ticker.C:
				if err := s.Comment("heartbeat"); err != nil {
					_ = s.Close()
					return
				}
			}
		}
	}()
}

// Send writes and flushes one event.
func (s *Stream) Send(event Event) error {
	if s == nil || s.writer == nil {
		return fmt.Errorf("lazysse: stream is not started")
	}
	select {
	case <-s.Done():
		return s.Context().Err()
	default:
	}

	frame, err := formatEvent(event)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.writer.Write([]byte(frame)); err != nil {
		return fmt.Errorf("lazysse: write event: %w", err)
	}
	if err := http.NewResponseController(s.writer).Flush(); err != nil {
		return fmt.Errorf("lazysse: flush event: %w", err)
	}
	return nil
}

// JSON marshals value and sends it as event data.
func (s *Stream) JSON(name string, value any) error {
	body, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("lazysse: marshal JSON event: %w", err)
	}
	return s.Send(Event{
		Event: name,
		Data:  []string{string(body)},
	})
}

// Comment sends an SSE comment.
func (s *Stream) Comment(text string) error {
	return s.Send(Event{
		Comment: []string{text},
	})
}

// Subscribe forwards events from source until the subscription or stream ends.
func (s *Stream) Subscribe(source Source, opts SubscribeOptions) error {
	if source == nil {
		return fmt.Errorf("lazysse: subscription source is nil")
	}
	if opts.LastEventID == "" {
		if id, ok := s.LastEventID(); ok {
			opts.LastEventID = id
		}
	}
	subscription, err := source.Subscribe(s.Context(), opts)
	if err != nil {
		return err
	}
	if subscription == nil {
		return fmt.Errorf("lazysse: subscription is nil")
	}
	defer subscription.Close()

	events := subscription.Events()
	for {
		select {
		case <-s.Done():
			return nil
		case event, ok := <-events:
			if !ok {
				return nil
			}
			if err := s.Send(event); err != nil {
				return err
			}
		}
	}
}

type streamStarter interface {
	StartStream(int) (http.ResponseWriter, error)
}

type responseUnwrapper interface {
	Unwrap() http.ResponseWriter
}

type flushErrorer interface {
	FlushError() error
}

type requestContextKey struct{}

func requestFromContext(ctx context.Context) (*http.Request, bool) {
	request, ok := ctx.Value(requestContextKey{}).(*http.Request)
	return request, ok && request != nil
}

func withRequest(ctx context.Context, request *http.Request) context.Context {
	return context.WithValue(ctx, requestContextKey{}, request)
}

func startStream(w http.ResponseWriter, status int) (http.ResponseWriter, error) {
	if starter, ok := w.(streamStarter); ok {
		return starter.StartStream(status)
	}
	if unwrapper, ok := w.(responseUnwrapper); ok {
		next := unwrapper.Unwrap()
		if next != nil && next != w {
			return startStream(next, status)
		}
	}
	w.WriteHeader(status)
	return w, nil
}

func supportsFlush(w http.ResponseWriter) bool {
	if w == nil {
		return false
	}
	if _, ok := w.(http.Flusher); ok {
		return true
	}
	if _, ok := w.(flushErrorer); ok {
		return true
	}
	if unwrapper, ok := w.(responseUnwrapper); ok {
		next := unwrapper.Unwrap()
		if next != nil && next != w {
			return supportsFlush(next)
		}
	}
	return false
}

func formatEvent(event Event) (string, error) {
	if hasNewline(event.Event) {
		return "", fmt.Errorf("lazysse: event name contains a newline")
	}
	if hasNewline(event.ID) {
		return "", fmt.Errorf("lazysse: event id contains a newline")
	}
	if event.Retry < 0 {
		return "", fmt.Errorf("lazysse: retry cannot be negative")
	}

	var builder strings.Builder
	for _, comment := range event.Comment {
		for _, line := range splitLines(comment) {
			builder.WriteString(": ")
			builder.WriteString(line)
			builder.WriteByte('\n')
		}
	}
	if event.Event != "" {
		builder.WriteString("event: ")
		builder.WriteString(event.Event)
		builder.WriteByte('\n')
	}
	if event.ID != "" {
		builder.WriteString("id: ")
		builder.WriteString(event.ID)
		builder.WriteByte('\n')
	}
	if event.Retry > 0 {
		builder.WriteString("retry: ")
		builder.WriteString(fmt.Sprint(event.Retry.Milliseconds()))
		builder.WriteByte('\n')
	}
	for _, data := range event.Data {
		for _, line := range splitLines(data) {
			builder.WriteString("data: ")
			builder.WriteString(line)
			builder.WriteByte('\n')
		}
	}
	builder.WriteByte('\n')
	return builder.String(), nil
}

func hasNewline(value string) bool {
	return strings.ContainsAny(value, "\r\n")
}

func splitLines(value string) []string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	return strings.Split(value, "\n")
}
