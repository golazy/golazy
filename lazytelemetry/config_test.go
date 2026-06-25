package lazytelemetry

import (
	"testing"
	"time"
)

func TestLoadConfigReadsOTELEnvironment(t *testing.T) {
	t.Setenv("OTEL_SDK_DISABLED", "true")
	t.Setenv("OTEL_SERVICE_NAME", "sample")
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "service.namespace=golazy,deployment.environment=development")
	t.Setenv("OTEL_PROPAGATORS", "tracecontext,baggage")
	t.Setenv("OTEL_TRACES_EXPORTER", "otlp,console")
	t.Setenv("OTEL_METRICS_EXPORTER", "none")
	t.Setenv("OTEL_LOGS_EXPORTER", "otlp/stdout")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://collector:4318")
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "http://traces:4318")
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_HEADERS", "api-key=secret")
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_PROTOCOL", "grpc")
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "http/protobuf")
	t.Setenv("OTEL_EXPORTER_OTLP_TIMEOUT", "10s")
	t.Setenv("OTEL_BSP_SCHEDULE_DELAY", "5000")
	t.Setenv("OTEL_BLRP_EXPORT_TIMEOUT", "30000")
	t.Setenv("OTEL_SPAN_ATTRIBUTE_COUNT_LIMIT", "64")
	t.Setenv("OTEL_LOGRECORD_ATTRIBUTE_COUNT_LIMIT", "32")
	t.Setenv("OTEL_METRICS_EXEMPLAR_FILTER", "trace_based")
	t.Setenv("OTEL_METRIC_EXPORT_INTERVAL", "60000")
	t.Setenv("OTEL_CONFIG_FILE", "otel.yaml")

	config, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if !config.SDKDisabled {
		t.Fatalf("SDKDisabled = false")
	}
	if config.ServiceName != "sample" {
		t.Fatalf("ServiceName = %q", config.ServiceName)
	}
	if config.ResourceAttributes != "service.namespace=golazy,deployment.environment=development" {
		t.Fatalf("ResourceAttributes = %q", config.ResourceAttributes)
	}
	if got, want := config.Propagators, []string{"tracecontext", "baggage"}; !stringSlicesEqual(got, want) {
		t.Fatalf("Propagators = %#v, want %#v", got, want)
	}
	if got, want := config.TracesExporter, []string{"otlp", "console"}; !stringSlicesEqual(got, want) {
		t.Fatalf("TracesExporter = %#v, want %#v", got, want)
	}
	if got, want := config.MetricsExporter, []string{"none"}; !stringSlicesEqual(got, want) {
		t.Fatalf("MetricsExporter = %#v, want %#v", got, want)
	}
	if got, want := config.LogsExporter, []string{"otlp/stdout"}; !stringSlicesEqual(got, want) {
		t.Fatalf("LogsExporter = %#v, want %#v", got, want)
	}
	if config.Exporter.OTLP.Endpoint != "http://collector:4318" {
		t.Fatalf("OTLP.Endpoint = %q", config.Exporter.OTLP.Endpoint)
	}
	if config.Exporter.OTLP.TracesEndpoint != "http://traces:4318" {
		t.Fatalf("OTLP.TracesEndpoint = %q", config.Exporter.OTLP.TracesEndpoint)
	}
	if config.Exporter.OTLP.TracesHeaders != "api-key=secret" {
		t.Fatalf("OTLP.TracesHeaders = %q", config.Exporter.OTLP.TracesHeaders)
	}
	if config.Exporter.OTLP.TracesProtocol != "grpc" {
		t.Fatalf("OTLP.TracesProtocol = %q", config.Exporter.OTLP.TracesProtocol)
	}
	if config.Exporter.OTLP.Protocol != "http/protobuf" {
		t.Fatalf("OTLP.Protocol = %q", config.Exporter.OTLP.Protocol)
	}
	if got, want := config.Exporter.OTLP.Timeout.Duration(), 10*time.Second; got != want {
		t.Fatalf("OTLP.Timeout = %s, want %s", got, want)
	}
	if got, want := config.BSP.ScheduleDelay.Duration(), 5*time.Second; got != want {
		t.Fatalf("BSP.ScheduleDelay = %s, want %s", got, want)
	}
	if got, want := config.BLRP.ExportTimeout.Duration(), 30*time.Second; got != want {
		t.Fatalf("BLRP.ExportTimeout = %s, want %s", got, want)
	}
	if config.SpanAttributeCountLimit != 64 {
		t.Fatalf("SpanAttributeCountLimit = %d", config.SpanAttributeCountLimit)
	}
	if config.LogrecordAttributeCountLimit != 32 {
		t.Fatalf("LogrecordAttributeCountLimit = %d", config.LogrecordAttributeCountLimit)
	}
	if config.MetricsExemplarFilter != "trace_based" {
		t.Fatalf("MetricsExemplarFilter = %q", config.MetricsExemplarFilter)
	}
	if got, want := config.MetricExportInterval.Duration(), time.Minute; got != want {
		t.Fatalf("MetricExportInterval = %s, want %s", got, want)
	}
	if config.ConfigFile != "otel.yaml" {
		t.Fatalf("ConfigFile = %q", config.ConfigFile)
	}
}

func TestDurationParsesGoDurationAndMilliseconds(t *testing.T) {
	for _, test := range []struct {
		raw  string
		want time.Duration
	}{
		{raw: "2500", want: 2500 * time.Millisecond},
		{raw: "2s", want: 2 * time.Second},
		{raw: "", want: 0},
	} {
		var got Duration
		if err := got.UnmarshalText([]byte(test.raw)); err != nil {
			t.Fatalf("UnmarshalText(%q): %v", test.raw, err)
		}
		if got.Duration() != test.want {
			t.Fatalf("UnmarshalText(%q) = %s, want %s", test.raw, got.Duration(), test.want)
		}
	}
}

func TestConfigJSONLogs(t *testing.T) {
	if (Config{LogsExporter: []string{"none"}}).JSONLogs() {
		t.Fatal("JSONLogs = true for none exporter")
	}
	if !(Config{LogsExporter: []string{"otlp"}}).JSONLogs() {
		t.Fatal("JSONLogs = false for otlp exporter")
	}
}

func TestConfigEnabledUsesOTELEnvironment(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		want   bool
	}{
		{name: "empty", config: Config{}, want: false},
		{name: "sdk disabled", config: Config{SDKDisabled: true, ServiceName: "sample"}, want: false},
		{name: "service name", config: Config{ServiceName: "sample"}, want: true},
		{name: "otlp traces exporter", config: Config{TracesExporter: []string{"otlp"}}, want: true},
		{name: "none traces exporter", config: Config{TracesExporter: []string{"none"}}, want: false},
		{name: "otlp endpoint", config: Config{Exporter: ExporterConfig{OTLP: OTLPExporterConfig{Endpoint: "http://127.0.0.1:4318"}}}, want: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.config.Enabled(); got != test.want {
				t.Fatalf("Enabled = %v, want %v", got, test.want)
			}
		})
	}
}

func TestConfigCaptureRequestFilesRequiresExporter(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		want   bool
	}{
		{name: "empty", config: Config{}, want: false},
		{name: "sdk disabled", config: Config{SDKDisabled: true, TracesExporter: []string{"otlp"}}, want: false},
		{name: "service name only", config: Config{ServiceName: "sample"}, want: false},
		{name: "traces exporter", config: Config{TracesExporter: []string{"otlp"}}, want: true},
		{name: "logs exporter", config: Config{LogsExporter: []string{"otlp"}}, want: true},
		{name: "metrics exporter", config: Config{MetricsExporter: []string{"otlp"}}, want: true},
		{name: "prometheus metrics exporter", config: Config{MetricsExporter: []string{"prometheus"}}, want: true},
		{name: "none exporter", config: Config{TracesExporter: []string{"none"}, LogsExporter: []string{"none"}}, want: false},
		{name: "otlp protocol", config: Config{Exporter: ExporterConfig{OTLP: OTLPExporterConfig{Protocol: "http/protobuf"}}}, want: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.config.captureRequestFiles(); got != test.want {
				t.Fatalf("captureRequestFiles = %v, want %v", got, test.want)
			}
		})
	}
}

func TestConfigPrometheusMetrics(t *testing.T) {
	if !(Config{MetricsExporter: []string{"prometheus"}}).PrometheusMetrics() {
		t.Fatal("PrometheusMetrics = false for prometheus exporter")
	}
	if (Config{SDKDisabled: true, MetricsExporter: []string{"prometheus"}}).PrometheusMetrics() {
		t.Fatal("PrometheusMetrics = true when SDK is disabled")
	}
	if (Config{MetricsExporter: []string{"otlp"}}).PrometheusMetrics() {
		t.Fatal("PrometheusMetrics = true for otlp exporter")
	}
}

func stringSlicesEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
