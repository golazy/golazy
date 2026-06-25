package lazymetrics

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRegistryRecordsCounterGaugeAndHistogram(t *testing.T) {
	registry := NewRegistry()

	requests := registry.NewCounter("requests_total", "method", "status")
	inflight := registry.NewGauge("requests_inflight", "method")
	duration := registry.NewHistogram("request_duration_seconds", "method")

	requests.WithLabelValues("GET", "200").Inc()
	requests.With(Labels{"method": "GET", "status": "200"}).Add(2)
	inflight.WithLabelValues("GET").Set(3)
	inflight.WithLabelValues("GET").Add(-1)
	duration.WithLabelValues("GET").Observe(0.25)
	duration.WithLabelValues("GET").Observe(0.75)

	snapshot := registry.Snapshot()
	if got := findMetric(snapshot.Counters, "requests_total", Labels{"method": "GET", "status": "200"}); got != 3 {
		t.Fatalf("counter = %v, want 3", got)
	}
	if got := findMetric(snapshot.Gauges, "requests_inflight", Labels{"method": "GET"}); got != 2 {
		t.Fatalf("gauge = %v, want 2", got)
	}
	histogram := findHistogram(snapshot.Histograms, "request_duration_seconds", Labels{"method": "GET"})
	if histogram.Count != 2 || histogram.Sum != 1.0 || histogram.Min != 0.25 || histogram.Max != 0.75 {
		t.Fatalf("histogram = %#v", histogram)
	}
	if got := histogramBucket(histogram, 0.25); got != 1 {
		t.Fatalf("0.25 bucket = %d, want 1", got)
	}
	if got := histogramBucket(histogram, 1); got != 2 {
		t.Fatalf("1 bucket = %d, want 2", got)
	}
}

func TestWritePrometheus(t *testing.T) {
	registry := NewRegistry()

	registry.NewCounter("requests_total", "method").WithLabelValues("GET").Inc()
	registry.NewHistogram("request_duration_seconds", "method").WithLabelValues("GET").Observe(0.25)

	var out bytes.Buffer
	if err := WritePrometheus(&out, registry.Snapshot()); err != nil {
		t.Fatal(err)
	}
	body := out.String()
	for _, want := range []string{
		"# TYPE requests_total counter\n",
		`requests_total{method="GET"} 1` + "\n",
		"# TYPE request_duration_seconds histogram\n",
		`request_duration_seconds_bucket{le="0.25",method="GET"} 1` + "\n",
		`request_duration_seconds_bucket{le="+Inf",method="GET"} 1` + "\n",
		`request_duration_seconds_sum{method="GET"} 0.25` + "\n",
		`request_duration_seconds_count{method="GET"} 1` + "\n",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("Prometheus output missing %q in:\n%s", want, body)
		}
	}
}

func TestWithLabelsMergesContextLabels(t *testing.T) {
	ctx := WithLabels(context.Background(), Labels{"method": "GET"})
	ctx = WithLabels(ctx, Labels{"status": "200"})

	labels := LabelsFromContext(ctx)
	if labels["method"] != "GET" || labels["status"] != "200" {
		t.Fatalf("labels = %#v", labels)
	}
	labels["method"] = "POST"
	if got := LabelsFromContext(ctx)["method"]; got != "GET" {
		t.Fatalf("context labels mutated to %q", got)
	}
}

func findMetric(metrics []MetricSnapshot, name string, labels Labels) float64 {
	for _, metric := range metrics {
		if metric.Name == name && sameLabels(metric.Labels, labels) {
			return metric.Value
		}
	}
	return 0
}

func findHistogram(metrics []HistogramSnapshot, name string, labels Labels) HistogramSnapshot {
	for _, metric := range metrics {
		if metric.Name == name && sameLabels(metric.Labels, labels) {
			return metric
		}
	}
	return HistogramSnapshot{}
}

func histogramBucket(histogram HistogramSnapshot, boundary float64) int64 {
	for _, bucket := range histogram.Buckets {
		if bucket.Le == boundary {
			return bucket.Count
		}
	}
	return 0
}

func sameLabels(left, right Labels) bool {
	if len(left) != len(right) {
		return false
	}
	for name, value := range left {
		if right[name] != value {
			return false
		}
	}
	return true
}
