package lazycache

import (
	"fmt"
	"io"
)

// PrometheusCollector returns a collector for cache statistics.
func PrometheusCollector(cache *Cache) func(io.Writer) error {
	return func(w io.Writer) error {
		return WritePrometheus(w, cache)
	}
}

// WritePrometheus writes cache statistics using the Prometheus text exposition
// format.
func WritePrometheus(w io.Writer, cache *Cache) error {
	stats := cache.Stats()
	enabled := 0
	if cache.Enabled() {
		enabled = 1
	}
	metrics := []struct {
		name  string
		typ   string
		value any
	}{
		{name: "golazy_cache_enabled", typ: "gauge", value: enabled},
		{name: "golazy_cache_entries", typ: "gauge", value: stats.Entries},
		{name: "golazy_cache_max_entries", typ: "gauge", value: stats.MaxEntries},
		{name: "golazy_cache_hits_total", typ: "counter", value: stats.Hits},
		{name: "golazy_cache_misses_total", typ: "counter", value: stats.Misses},
		{name: "golazy_cache_sets_total", typ: "counter", value: stats.Sets},
		{name: "golazy_cache_evictions_total", typ: "counter", value: stats.Evictions},
	}
	for _, metric := range metrics {
		if _, err := fmt.Fprintf(w, "# TYPE %s %s\n%s %v\n", metric.name, metric.typ, metric.name, metric.value); err != nil {
			return err
		}
	}
	return nil
}
