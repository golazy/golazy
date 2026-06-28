package lazylogs

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"golazy.dev/lazytelemetry/lazytracing"
)

func TestContextLoggerAddsAttrsGroupsAndTags(t *testing.T) {
	var out bytes.Buffer
	ctx := WithLogger(context.Background(), NewJSONLogger(&out))
	ctx = WithAttrs(ctx, slog.String("request_id", "req-1"))
	ctx = WithTags(ctx, "http", " controller ")
	ctx = WithGroup(ctx, "controller")

	Info(ctx, "loaded", slog.String("name", "home"))

	line := out.String()
	for _, want := range []string{
		`"request_id":"req-1"`,
		`"tags":["http","controller"]`,
		`"controller":{"name":"home"}`,
		`"msg":"loaded"`,
	} {
		if !strings.Contains(line, want) {
			t.Fatalf("log line %q does not contain %q", line, want)
		}
	}
}

func TestLogAttrsRecordsSpanEvent(t *testing.T) {
	var out bytes.Buffer
	ctx := WithLogger(context.Background(), NewJSONLogger(&out))
	ctx = lazytracing.WithRequestID(ctx, "req-1")
	ctx, span := lazytracing.StartSpan(ctx, "request")
	defer span.End()

	Warn(ctx, "slow render", slog.String("view", "home/index"))

	events := span.Events()
	if len(events) != 1 {
		t.Fatalf("events = %#v", events)
	}
	if events[0].Name != "log" {
		t.Fatalf("event name = %q", events[0].Name)
	}
	if got := attrString(events[0].Attributes, "message"); got != "slow render" {
		t.Fatalf("message attr = %q", got)
	}
	if got := attrString(events[0].Attributes, "request_id"); got != "req-1" {
		t.Fatalf("request_id attr = %q", got)
	}
	if got := attrString(events[0].Attributes, "view"); got != "home/index" {
		t.Fatalf("view attr = %q", got)
	}
}

func attrString(attrs []slog.Attr, name string) string {
	for _, attr := range attrs {
		if attr.Key == name {
			return attr.Value.String()
		}
	}
	return ""
}
