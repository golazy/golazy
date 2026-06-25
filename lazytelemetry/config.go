// Package lazytelemetry configures GoLazy telemetry hooks.
package lazytelemetry

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"golazy.dev/lazyconfig"
)

// Config contains OpenTelemetry-compatible environment configuration for
// GoLazy telemetry.
type Config struct {
	SDKDisabled        bool
	Entities           string
	ResourceAttributes string
	ServiceName        string
	LogLevel           string
	Propagators        []string `var:"PROPAGATORS"`
	TracesSampler      string
	TracesSamplerArg   string

	BSP  BatchSpanProcessorConfig
	BLRP BatchLogRecordProcessorConfig

	AttributeValueLengthLimit int
	AttributeCountLimit       int

	SpanAttributeValueLengthLimit int
	SpanAttributeCountLimit       int
	SpanEventCountLimit           int
	SpanLinkCountLimit            int
	EventAttributeCountLimit      int
	LinkAttributeCountLimit       int

	LogrecordAttributeValueLengthLimit int `var:"LOGRECORD_ATTRIBUTE_VALUE_LENGTH_LIMIT"`
	LogrecordAttributeCountLimit       int `var:"LOGRECORD_ATTRIBUTE_COUNT_LIMIT"`

	TracesExporter  []string
	MetricsExporter []string
	LogsExporter    []string

	Exporter ExporterConfig

	MetricsExemplarFilter string
	MetricExportInterval  Duration
	MetricExportTimeout   Duration

	ConfigFile             string
	ExperimentalConfigFile string
}

// BatchSpanProcessorConfig contains OTEL_BSP_* values.
type BatchSpanProcessorConfig struct {
	ScheduleDelay      Duration
	ExportTimeout      Duration
	MaxQueueSize       int
	MaxExportBatchSize int
}

// BatchLogRecordProcessorConfig contains OTEL_BLRP_* values.
type BatchLogRecordProcessorConfig struct {
	ScheduleDelay      Duration
	ExportTimeout      Duration
	MaxQueueSize       int
	MaxExportBatchSize int
}

// ExporterConfig contains OpenTelemetry exporter configuration.
type ExporterConfig struct {
	OTLP       OTLPExporterConfig
	Zipkin     ZipkinExporterConfig
	Prometheus PrometheusExporterConfig
}

// OTLPExporterConfig contains OTEL_EXPORTER_OTLP_* values.
type OTLPExporterConfig struct {
	Endpoint        string
	TracesEndpoint  string
	MetricsEndpoint string
	LogsEndpoint    string

	Insecure        bool
	TracesInsecure  bool
	MetricsInsecure bool
	LogsInsecure    bool

	Certificate        string
	TracesCertificate  string
	MetricsCertificate string
	LogsCertificate    string

	ClientKey        string
	TracesClientKey  string
	MetricsClientKey string
	LogsClientKey    string

	ClientCertificate        string
	TracesClientCertificate  string
	MetricsClientCertificate string
	LogsClientCertificate    string

	Headers        string
	TracesHeaders  string
	MetricsHeaders string
	LogsHeaders    string

	Compression        string
	TracesCompression  string
	MetricsCompression string
	LogsCompression    string

	Timeout        Duration
	TracesTimeout  Duration
	MetricsTimeout Duration
	LogsTimeout    Duration

	Protocol        string
	TracesProtocol  string
	MetricsProtocol string
	LogsProtocol    string

	SpanInsecure   bool
	MetricInsecure bool
}

// ZipkinExporterConfig contains OTEL_EXPORTER_ZIPKIN_* values.
type ZipkinExporterConfig struct {
	Endpoint string
	Timeout  Duration
	Protocol string
}

// PrometheusExporterConfig contains OTEL_EXPORTER_PROMETHEUS_* values.
type PrometheusExporterConfig struct {
	Host string
	Port int
}

// Duration stores an OTEL duration value.
//
// OTEL SDK values are commonly represented as milliseconds, while OTLP exporter
// timeout values are often written as Go-style durations such as "10s". Duration
// accepts both forms.
type Duration time.Duration

// UnmarshalText implements encoding.TextUnmarshaler.
func (d *Duration) UnmarshalText(data []byte) error {
	raw := strings.TrimSpace(string(data))
	if raw == "" {
		*d = 0
		return nil
	}
	parsed, err := time.ParseDuration(raw)
	if err == nil {
		*d = Duration(parsed)
		return nil
	}
	milliseconds, parseErr := strconv.ParseInt(raw, 10, 64)
	if parseErr != nil {
		return fmt.Errorf("parse %q as duration or milliseconds: %w", raw, parseErr)
	}
	*d = Duration(time.Duration(milliseconds) * time.Millisecond)
	return nil
}

// Duration returns d as a time.Duration.
func (d Duration) Duration() time.Duration {
	return time.Duration(d)
}

// LoadConfig reads Config from the process environment.
func LoadConfig() (Config, error) {
	return lazyconfig.Getenv[Config](lazyconfig.RemoveEnvNamePrefix("OTEL"))
}

// MustLoadConfig reads Config and panics when the environment is invalid.
func MustLoadConfig() Config {
	return lazyconfig.MustGetenv[Config](lazyconfig.RemoveEnvNamePrefix("OTEL"))
}
