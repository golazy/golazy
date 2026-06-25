// Package lazymetrics provides lightweight metric helpers for GoLazy telemetry.
package lazymetrics

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
)

type labelsContextKey struct{}

// Labels stores low-cardinality metric labels.
type Labels map[string]string

// Registry stores in-memory metrics.
type Registry struct {
	mu sync.Mutex

	counters   map[metricKey]float64
	gauges     map[metricKey]float64
	histograms map[metricKey]histogramState
}

type metricKey struct {
	name   string
	labels string
}

type histogramState struct {
	count   int64
	sum     float64
	min     float64
	max     float64
	buckets map[float64]int64
}

// MetricSnapshot is a point-in-time counter or gauge value.
type MetricSnapshot struct {
	Name   string
	Labels Labels
	Value  float64
}

// HistogramSnapshot is a point-in-time histogram summary.
type HistogramSnapshot struct {
	Name    string
	Labels  Labels
	Count   int64
	Sum     float64
	Min     float64
	Max     float64
	Buckets []HistogramBucketSnapshot
}

// HistogramBucketSnapshot is a cumulative histogram bucket.
type HistogramBucketSnapshot struct {
	Le    float64
	Count int64
}

// Snapshot contains all metrics in a registry.
type Snapshot struct {
	Counters   []MetricSnapshot
	Gauges     []MetricSnapshot
	Histograms []HistogramSnapshot
}

// NewRegistry creates an empty metric registry.
func NewRegistry() *Registry {
	return &Registry{
		counters:   map[metricKey]float64{},
		gauges:     map[metricKey]float64{},
		histograms: map[metricKey]histogramState{},
	}
}

// CounterVec is a named counter with declared label names.
type CounterVec struct {
	registry   *Registry
	name       string
	labelNames []string
}

// GaugeVec is a named gauge with declared label names.
type GaugeVec struct {
	registry   *Registry
	name       string
	labelNames []string
}

// HistogramVec is a named histogram with declared label names.
type HistogramVec struct {
	registry   *Registry
	name       string
	labelNames []string
}

// Counter is a counter metric handle.
type Counter struct {
	registry *Registry
	name     string
	labels   Labels
}

// Gauge is a gauge metric handle.
type Gauge struct {
	registry *Registry
	name     string
	labels   Labels
}

// Histogram is a histogram metric handle.
type Histogram struct {
	registry *Registry
	name     string
	labels   Labels
}

// NewCounter creates a counter vector.
func (r *Registry) NewCounter(name string, labelNames ...string) *CounterVec {
	return &CounterVec{registry: r, name: name, labelNames: cleanNames(labelNames)}
}

// NewGauge creates a gauge vector.
func (r *Registry) NewGauge(name string, labelNames ...string) *GaugeVec {
	return &GaugeVec{registry: r, name: name, labelNames: cleanNames(labelNames)}
}

// NewHistogram creates a histogram vector.
func (r *Registry) NewHistogram(name string, labelNames ...string) *HistogramVec {
	return &HistogramVec{registry: r, name: name, labelNames: cleanNames(labelNames)}
}

// WithLabelValues returns a counter for values in label-name order.
func (v *CounterVec) WithLabelValues(values ...string) Counter {
	return Counter{registry: v.registry, name: v.name, labels: labelsFromValues(v.labelNames, values)}
}

// With returns a counter for labels.
func (v *CounterVec) With(labels Labels) Counter {
	return Counter{registry: v.registry, name: v.name, labels: filterLabels(v.labelNames, labels)}
}

// WithLabelValues returns a gauge for values in label-name order.
func (v *GaugeVec) WithLabelValues(values ...string) Gauge {
	return Gauge{registry: v.registry, name: v.name, labels: labelsFromValues(v.labelNames, values)}
}

// With returns a gauge for labels.
func (v *GaugeVec) With(labels Labels) Gauge {
	return Gauge{registry: v.registry, name: v.name, labels: filterLabels(v.labelNames, labels)}
}

// WithLabelValues returns a histogram for values in label-name order.
func (v *HistogramVec) WithLabelValues(values ...string) Histogram {
	return Histogram{registry: v.registry, name: v.name, labels: labelsFromValues(v.labelNames, values)}
}

// With returns a histogram for labels.
func (v *HistogramVec) With(labels Labels) Histogram {
	return Histogram{registry: v.registry, name: v.name, labels: filterLabels(v.labelNames, labels)}
}

// Inc increments the counter by one.
func (c Counter) Inc() {
	c.Add(1)
}

// Add increments the counter by value. Negative values are ignored.
func (c Counter) Add(value float64) {
	if c.registry == nil || value < 0 {
		return
	}
	c.registry.mu.Lock()
	defer c.registry.mu.Unlock()
	c.registry.ensure()
	c.registry.counters[key(c.name, c.labels)] += value
}

// Add changes the gauge by value.
func (g Gauge) Add(value float64) {
	if g.registry == nil {
		return
	}
	g.registry.mu.Lock()
	defer g.registry.mu.Unlock()
	g.registry.ensure()
	g.registry.gauges[key(g.name, g.labels)] += value
}

// Set sets the gauge value.
func (g Gauge) Set(value float64) {
	if g.registry == nil {
		return
	}
	g.registry.mu.Lock()
	defer g.registry.mu.Unlock()
	g.registry.ensure()
	g.registry.gauges[key(g.name, g.labels)] = value
}

// Observe records a histogram observation.
func (h Histogram) Observe(value float64) {
	if h.registry == nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return
	}
	h.registry.mu.Lock()
	defer h.registry.mu.Unlock()
	h.registry.ensure()
	k := key(h.name, h.labels)
	state := h.registry.histograms[k]
	if state.count == 0 {
		state.min = value
		state.max = value
		state.buckets = map[float64]int64{}
	} else {
		state.min = math.Min(state.min, value)
		state.max = math.Max(state.max, value)
	}
	if state.buckets == nil {
		state.buckets = map[float64]int64{}
	}
	for _, bucket := range defaultHistogramBuckets {
		if value <= bucket {
			state.buckets[bucket]++
		}
	}
	state.count++
	state.sum += value
	h.registry.histograms[k] = state
}

// Snapshot returns a copy of all registry values.
func (r *Registry) Snapshot() Snapshot {
	if r == nil {
		return Snapshot{}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	var snapshot Snapshot
	for key, value := range r.counters {
		snapshot.Counters = append(snapshot.Counters, MetricSnapshot{
			Name:   key.name,
			Labels: decodeLabels(key.labels),
			Value:  value,
		})
	}
	for key, value := range r.gauges {
		snapshot.Gauges = append(snapshot.Gauges, MetricSnapshot{
			Name:   key.name,
			Labels: decodeLabels(key.labels),
			Value:  value,
		})
	}
	for key, value := range r.histograms {
		snapshot.Histograms = append(snapshot.Histograms, HistogramSnapshot{
			Name:    key.name,
			Labels:  decodeLabels(key.labels),
			Count:   value.count,
			Sum:     value.sum,
			Min:     value.min,
			Max:     value.max,
			Buckets: histogramBuckets(value),
		})
	}
	sortMetricSnapshots(snapshot.Counters)
	sortMetricSnapshots(snapshot.Gauges)
	sortHistogramSnapshots(snapshot.Histograms)
	return snapshot
}

// WithLabels attaches metric labels to ctx.
func WithLabels(ctx context.Context, labels Labels) context.Context {
	if len(labels) == 0 {
		return ctx
	}
	merged := LabelsFromContext(ctx)
	for name, value := range labels {
		name = strings.TrimSpace(name)
		if name != "" {
			merged[name] = strings.TrimSpace(value)
		}
	}
	return context.WithValue(ctx, labelsContextKey{}, merged)
}

// LabelsFromContext returns metric labels attached to ctx.
func LabelsFromContext(ctx context.Context) Labels {
	if labels, ok := ctx.Value(labelsContextKey{}).(Labels); ok {
		return cloneLabels(labels)
	}
	return Labels{}
}

func (r *Registry) ensure() {
	if r.counters == nil {
		r.counters = map[metricKey]float64{}
	}
	if r.gauges == nil {
		r.gauges = map[metricKey]float64{}
	}
	if r.histograms == nil {
		r.histograms = map[metricKey]histogramState{}
	}
}

func key(name string, labels Labels) metricKey {
	return metricKey{name: strings.TrimSpace(name), labels: encodeLabels(labels)}
}

func cleanNames(names []string) []string {
	clean := make([]string, 0, len(names))
	seen := map[string]bool{}
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		clean = append(clean, name)
	}
	return clean
}

func labelsFromValues(names []string, values []string) Labels {
	labels := Labels{}
	for index, name := range names {
		if index >= len(values) {
			break
		}
		labels[name] = strings.TrimSpace(values[index])
	}
	return labels
}

func filterLabels(names []string, labels Labels) Labels {
	if len(names) == 0 {
		return cloneLabels(labels)
	}
	out := Labels{}
	for _, name := range names {
		if value, ok := labels[name]; ok {
			out[name] = strings.TrimSpace(value)
		}
	}
	return out
}

func cloneLabels(labels Labels) Labels {
	out := Labels{}
	for name, value := range labels {
		name = strings.TrimSpace(name)
		if name != "" {
			out[name] = strings.TrimSpace(value)
		}
	}
	return out
}

func encodeLabels(labels Labels) string {
	if len(labels) == 0 {
		return ""
	}
	names := make([]string, 0, len(labels))
	for name := range labels {
		name = strings.TrimSpace(name)
		if name != "" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	var builder strings.Builder
	for index, name := range names {
		if index > 0 {
			builder.WriteByte('\n')
		}
		builder.WriteString(name)
		builder.WriteByte('=')
		builder.WriteString(strings.TrimSpace(labels[name]))
	}
	return builder.String()
}

func decodeLabels(encoded string) Labels {
	labels := Labels{}
	if encoded == "" {
		return labels
	}
	for _, part := range strings.Split(encoded, "\n") {
		name, value, ok := strings.Cut(part, "=")
		if ok && name != "" {
			labels[name] = value
		}
	}
	return labels
}

func sortMetricSnapshots(snapshots []MetricSnapshot) {
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshotKey(snapshots[i].Name, snapshots[i].Labels) < snapshotKey(snapshots[j].Name, snapshots[j].Labels)
	})
}

func sortHistogramSnapshots(snapshots []HistogramSnapshot) {
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshotKey(snapshots[i].Name, snapshots[i].Labels) < snapshotKey(snapshots[j].Name, snapshots[j].Labels)
	})
}

var defaultHistogramBuckets = []float64{
	0.005,
	0.01,
	0.025,
	0.05,
	0.1,
	0.25,
	0.5,
	1,
	2.5,
	5,
	10,
}

func histogramBuckets(state histogramState) []HistogramBucketSnapshot {
	buckets := make([]HistogramBucketSnapshot, 0, len(defaultHistogramBuckets))
	for _, boundary := range defaultHistogramBuckets {
		buckets = append(buckets, HistogramBucketSnapshot{
			Le:    boundary,
			Count: state.buckets[boundary],
		})
	}
	return buckets
}

func snapshotKey(name string, labels Labels) string {
	return fmt.Sprintf("%s\n%s", name, encodeLabels(labels))
}
