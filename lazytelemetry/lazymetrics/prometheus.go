package lazymetrics

import (
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

// PrometheusCollector writes additional Prometheus text exposition metrics.
type PrometheusCollector func(io.Writer) error

// PrometheusHandler returns an HTTP handler for Prometheus text exposition.
func PrometheusHandler(registry *Registry, collectors ...PrometheusCollector) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		if err := WritePrometheus(w, registry.Snapshot()); err != nil {
			http.Error(w, fmt.Sprintf("metrics: %v", err), http.StatusInternalServerError)
			return
		}
		for _, collector := range collectors {
			if collector == nil {
				continue
			}
			if err := collector(w); err != nil {
				http.Error(w, fmt.Sprintf("metrics: %v", err), http.StatusInternalServerError)
				return
			}
		}
	})
}

// WritePrometheus writes snapshot using the Prometheus text exposition format.
func WritePrometheus(w io.Writer, snapshot Snapshot) error {
	if err := writePrometheusMetrics(w, "counter", snapshot.Counters); err != nil {
		return err
	}
	if err := writePrometheusMetrics(w, "gauge", snapshot.Gauges); err != nil {
		return err
	}
	for _, histogram := range snapshot.Histograms {
		if _, err := fmt.Fprintf(w, "# TYPE %s histogram\n", prometheusName(histogram.Name)); err != nil {
			return err
		}
		for _, bucket := range histogram.Buckets {
			labels := cloneLabels(histogram.Labels)
			labels["le"] = prometheusFloat(bucket.Le)
			if _, err := fmt.Fprintf(w, "%s_bucket%s %d\n", prometheusName(histogram.Name), prometheusLabels(labels), bucket.Count); err != nil {
				return err
			}
		}
		labels := cloneLabels(histogram.Labels)
		labels["le"] = "+Inf"
		if _, err := fmt.Fprintf(w, "%s_bucket%s %d\n", prometheusName(histogram.Name), prometheusLabels(labels), histogram.Count); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "%s_sum%s %s\n", prometheusName(histogram.Name), prometheusLabels(histogram.Labels), prometheusFloat(histogram.Sum)); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "%s_count%s %d\n", prometheusName(histogram.Name), prometheusLabels(histogram.Labels), histogram.Count); err != nil {
			return err
		}
	}
	return nil
}

func writePrometheusMetrics(w io.Writer, typ string, metrics []MetricSnapshot) error {
	for _, metric := range metrics {
		name := prometheusName(metric.Name)
		if _, err := fmt.Fprintf(w, "# TYPE %s %s\n", name, typ); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "%s%s %s\n", name, prometheusLabels(metric.Labels), prometheusFloat(metric.Value)); err != nil {
			return err
		}
	}
	return nil
}

func prometheusName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "golazy_metric"
	}
	var builder strings.Builder
	lastUnderscore := false
	for index, r := range name {
		valid := r == '_' || r == ':' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || index > 0 && r >= '0' && r <= '9'
		if valid {
			builder.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			builder.WriteByte('_')
			lastUnderscore = true
		}
	}
	out := strings.Trim(builder.String(), "_")
	if out == "" {
		return "golazy_metric"
	}
	return out
}

func prometheusLabels(labels Labels) string {
	if len(labels) == 0 {
		return ""
	}
	values := map[string]string{}
	names := make([]string, 0, len(labels))
	for name, value := range labels {
		name = prometheusName(name)
		if name == "" {
			continue
		}
		if _, ok := values[name]; !ok {
			names = append(names, name)
		}
		values[name] = value
	}
	sort.Strings(names)
	var builder strings.Builder
	builder.WriteByte('{')
	for index, name := range names {
		if index > 0 {
			builder.WriteByte(',')
		}
		builder.WriteString(name)
		builder.WriteString(`="`)
		builder.WriteString(prometheusLabelValue(values[name]))
		builder.WriteByte('"')
	}
	builder.WriteByte('}')
	return builder.String()
}

func prometheusLabelValue(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, "\n", `\n`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return value
}

func prometheusFloat(value float64) string {
	switch {
	case math.IsInf(value, 1):
		return "+Inf"
	case math.IsInf(value, -1):
		return "-Inf"
	default:
		return strconv.FormatFloat(value, 'g', -1, 64)
	}
}
