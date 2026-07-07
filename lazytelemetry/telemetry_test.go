package lazytelemetry

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestNewAttachesTelemetryToContext(t *testing.T) {
	t.Setenv("OTEL_SERVICE_NAME", "sample")

	telemetry, ctx, err := New(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if telemetry == nil {
		t.Fatal("telemetry is nil")
	}
	got, ok := FromContext(ctx)
	if !ok {
		t.Fatal("telemetry missing from context")
	}
	if got != telemetry {
		t.Fatal("context telemetry does not match initialized telemetry")
	}
	if telemetry.Config().ServiceName != "sample" {
		t.Fatalf("service name = %q, want sample", telemetry.Config().ServiceName)
	}
	if err := telemetry.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
}

func TestNewInitializesConfiguredTraceExporter(t *testing.T) {
	t.Cleanup(func() {
		otel.SetTracerProvider(noop.NewTracerProvider())
	})
	t.Setenv("OTEL_TRACES_EXPORTER", "console")
	t.Setenv("OTEL_SERVICE_NAME", "sample")

	telemetry, _, err := New(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if telemetry == nil {
		t.Fatal("telemetry is nil")
	}
	if !telemetry.Config().traceExportEnabled() {
		t.Fatal("traceExportEnabled = false, want true")
	}
	if err := telemetry.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
}
