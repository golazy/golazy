package lazytracing

import (
	"context"
	"log/slog"
	"testing"
)

func TestParseTraceparent(t *testing.T) {
	traceContext, ok := ParseTraceparent(
		"00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
		"vendor=value",
	)
	if !ok {
		t.Fatal("traceparent was not parsed")
	}
	if traceContext.TraceID != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Fatalf("TraceID = %q", traceContext.TraceID)
	}
	if traceContext.SpanID != "00f067aa0ba902b7" {
		t.Fatalf("SpanID = %q", traceContext.SpanID)
	}
	if traceContext.TraceFlags != "01" {
		t.Fatalf("TraceFlags = %q", traceContext.TraceFlags)
	}
	if traceContext.TraceState != "vendor=value" {
		t.Fatalf("TraceState = %q", traceContext.TraceState)
	}
	if !traceContext.Remote {
		t.Fatal("Remote = false")
	}
}

func TestParseTraceparentRejectsInvalidValues(t *testing.T) {
	for _, value := range []string{
		"",
		"ff-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
		"00-00000000000000000000000000000000-00f067aa0ba902b7-01",
		"00-4bf92f3577b34da6a3ce929d0e0e4736-0000000000000000-01",
		"00-xyz92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
	} {
		if traceContext, ok := ParseTraceparent(value, ""); ok {
			t.Fatalf("ParseTraceparent(%q) = %#v, want invalid", value, traceContext)
		}
	}
}

func TestStartSpanUsesIncomingTraceContext(t *testing.T) {
	ctx := WithTraceContext(context.Background(), TraceContext{
		TraceID: "4bf92f3577b34da6a3ce929d0e0e4736",
		SpanID:  "00f067aa0ba902b7",
		Remote:  true,
	})

	ctx, span := StartSpan(ctx, "controller.action", slog.String("controller", "home"))
	defer span.End()

	if span.TraceID() != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Fatalf("TraceID = %q", span.TraceID())
	}
	if span.ParentID() != "00f067aa0ba902b7" {
		t.Fatalf("ParentID = %q", span.ParentID())
	}
	if span.SpanID() == "" || span.SpanID() == span.ParentID() {
		t.Fatalf("SpanID = %q, ParentID = %q", span.SpanID(), span.ParentID())
	}
	if SpanFromContext(ctx) != span {
		t.Fatal("span not attached to context")
	}

	span.AddEvent("render", slog.String("view", "home/index"))
	span.RecordError(assertErr("broken"))
	if len(span.Events()) != 2 {
		t.Fatalf("events = %#v", span.Events())
	}
	if span.Error() == nil {
		t.Fatal("span error was not recorded")
	}
}

type assertErr string

func (e assertErr) Error() string {
	return string(e)
}
