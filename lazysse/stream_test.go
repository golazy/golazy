package lazysse

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestStartConfiguresHeadersAndSendsEvents(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Last-Event-ID", "42")
	response := httptest.NewRecorder()

	stream, err := Start(
		response,
		request,
		Status(http.StatusAccepted),
		Header("X-Test", "yes"),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()

	if response.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusAccepted)
	}
	if got, want := response.Header().Get("Content-Type"), "text/event-stream"; got != want {
		t.Fatalf("Content-Type = %q, want %q", got, want)
	}
	if got, want := response.Header().Get("Cache-Control"), "no-cache, no-transform"; got != want {
		t.Fatalf("Cache-Control = %q, want %q", got, want)
	}
	if got, want := response.Header().Get("X-Test"), "yes"; got != want {
		t.Fatalf("X-Test = %q, want %q", got, want)
	}
	if got, ok := stream.LastEventID(); !ok || got != "42" {
		t.Fatalf("LastEventID() = %q, %v; want 42, true", got, ok)
	}

	err = stream.Send(Event{
		Comment: []string{"hello\nthere"},
		Event:   "update",
		ID:      "43",
		Retry:   2500 * time.Millisecond,
		Data:    []string{"first", "second\nthird"},
	})
	if err != nil {
		t.Fatal(err)
	}

	want := ": hello\n" +
		": there\n" +
		"event: update\n" +
		"id: 43\n" +
		"retry: 2500\n" +
		"data: first\n" +
		"data: second\n" +
		"data: third\n\n"
	if got := response.Body.String(); got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestJSONAndCommentHelpers(t *testing.T) {
	stream, response := startTestStream(t)
	defer stream.Close()

	if err := stream.JSON("message", map[string]string{"body": "hello"}); err != nil {
		t.Fatal(err)
	}
	if err := stream.Comment("keepalive"); err != nil {
		t.Fatal(err)
	}

	want := "event: message\n" +
		"data: {\"body\":\"hello\"}\n\n" +
		": keepalive\n\n"
	if got := response.Body.String(); got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestSendValidatesMetadata(t *testing.T) {
	tests := []struct {
		name  string
		event Event
		want  string
	}{
		{
			name:  "event newline",
			event: Event{Event: "bad\nevent"},
			want:  "event name contains a newline",
		},
		{
			name:  "id newline",
			event: Event{ID: "bad\nid"},
			want:  "event id contains a newline",
		},
		{
			name:  "negative retry",
			event: Event{Retry: -time.Second},
			want:  "retry cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stream, _ := startTestStream(t)
			defer stream.Close()
			err := stream.Send(tt.event)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want it to contain %q", err, tt.want)
			}
		})
	}
}

func TestStartRejectsUnsupportedWritersBeforeWriting(t *testing.T) {
	writer := &plainWriter{header: http.Header{}}
	_, err := Start(writer, httptest.NewRequest(http.MethodGet, "/", nil))
	if err == nil || !strings.Contains(err.Error(), "does not support flushing") {
		t.Fatalf("error = %v, want flushing error", err)
	}
	if writer.status != 0 {
		t.Fatalf("status = %d, want no write", writer.status)
	}
	if writer.body.String() != "" {
		t.Fatalf("body = %q, want empty", writer.body.String())
	}
}

func TestSubscribeForwardsSourceEvents(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Last-Event-ID", "7")
	response := httptest.NewRecorder()
	stream, err := Start(response, request)
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()

	source := &recordingSource{events: make(chan Event, 1)}
	source.events <- Event{Event: "message", Data: []string{"hello"}}
	close(source.events)

	if err := stream.Subscribe(source, SubscribeOptions{}); err != nil {
		t.Fatal(err)
	}
	if source.lastEventID != "7" {
		t.Fatalf("LastEventID = %q, want 7", source.lastEventID)
	}
	if !source.closed {
		t.Fatal("subscription was not closed")
	}
	if got, want := response.Body.String(), "event: message\ndata: hello\n\n"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestSubscribeRejectsNilSubscription(t *testing.T) {
	stream, _ := startTestStream(t)
	defer stream.Close()

	err := stream.Subscribe(nilSubscriptionSource{}, SubscribeOptions{})
	if err == nil || !strings.Contains(err.Error(), "subscription is nil") {
		t.Fatalf("error = %v, want nil subscription error", err)
	}
}

func startTestStream(t *testing.T) (*Stream, *httptest.ResponseRecorder) {
	t.Helper()
	response := httptest.NewRecorder()
	stream, err := Start(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatal(err)
	}
	return stream, response
}

type plainWriter struct {
	header http.Header
	body   strings.Builder
	status int
}

func (w *plainWriter) Header() http.Header {
	return w.header
}

func (w *plainWriter) Write(data []byte) (int, error) {
	return w.body.Write(data)
}

func (w *plainWriter) WriteHeader(status int) {
	w.status = status
}

type recordingSource struct {
	events      chan Event
	lastEventID string
	closed      bool
}

type nilSubscriptionSource struct{}

func (nilSubscriptionSource) Subscribe(context.Context, SubscribeOptions) (Subscription, error) {
	return nil, nil
}

func (s *recordingSource) Subscribe(_ context.Context, opts SubscribeOptions) (Subscription, error) {
	s.lastEventID = opts.LastEventID
	return s, nil
}

func (s *recordingSource) Events() <-chan Event {
	return s.events
}

func (s *recordingSource) Close() error {
	s.closed = true
	return nil
}
