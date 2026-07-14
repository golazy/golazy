package lazycache

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

type memoryBackend struct {
	values map[string]any
	stats  Stats
}

func (b *memoryBackend) Get(key string) (any, error) {
	if value, ok := b.values[key]; ok {
		b.stats.Hits++
		return value, nil
	}
	b.stats.Misses++
	return nil, ErrMiss
}

func (b *memoryBackend) Set(key string, value any) error {
	if b.values == nil {
		b.values = map[string]any{}
	}
	b.values[key] = value
	b.stats.Sets++
	b.stats.Entries = len(b.values)
	return nil
}

func (b *memoryBackend) Stats() Stats {
	return b.stats
}

func (b *memoryBackend) Keys() []string {
	keys := make([]string, 0, len(b.values))
	for key := range b.values {
		keys = append(keys, key)
	}
	return keys
}

func (b *memoryBackend) Entries() []EntryInfo {
	entries := make([]EntryInfo, 0, len(b.values))
	for key, value := range b.values {
		content := fmt.Sprint(value)
		entries = append(entries, EntryInfo{
			Key:            key,
			SizeBytes:      int64(len(content)),
			CreatedAt:      time.Unix(1, 0),
			UpdatedAt:      time.Unix(1, 0),
			LastAccessedAt: time.Unix(1, 0),
		})
	}
	return entries
}

func (b *memoryBackend) Entry(key string) (EntryDetail, error) {
	value, ok := b.values[key]
	if !ok {
		return EntryDetail{}, ErrMiss
	}
	content := fmt.Sprint(value)
	return EntryDetail{
		EntryInfo: EntryInfo{
			Key:            key,
			SizeBytes:      int64(len(content)),
			CreatedAt:      time.Unix(1, 0),
			UpdatedAt:      time.Unix(1, 0),
			LastAccessedAt: time.Unix(1, 0),
		},
		Content:     content,
		ContentType: "text/plain; charset=utf-8",
	}, nil
}

func TestNewRequiresBackend(t *testing.T) {
	if _, err := New(Options{}); err == nil {
		t.Fatal("New succeeded, want backend error")
	}
}

func TestKeyBuildsStableJoinedParts(t *testing.T) {
	stamp := time.Date(2026, 6, 25, 12, 30, 0, 42, time.FixedZone("CEST", 2*60*60))
	key, err := Key("post", 42, nil, stamp)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := key, "post-42--2026-06-25T10:30:00.000000042Z"; got != want {
		t.Fatalf("Key = %q, want %q", got, want)
	}

	var stampPointer *time.Time
	key, err = Key("post", stampPointer, "card")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := key, "post--card"; got != want {
		t.Fatalf("Key = %q, want %q", got, want)
	}
	key, err = Key(nil)
	if err != nil {
		t.Fatal(err)
	}
	if key != "" {
		t.Fatalf("Key(nil) = %q, want empty string", key)
	}

	for _, parts := range [][]any{{}, {""}} {
		if _, err := Key(parts...); err == nil {
			t.Fatalf("Key(%#v) succeeded, want error", parts)
		}
	}
}

func TestCacheGetSetAndTypedHelpers(t *testing.T) {
	cache, err := New(Options{Backend: &memoryBackend{}})
	if err != nil {
		t.Fatal(err)
	}
	if err := Set(cache, "Ada", "user", 1); err != nil {
		t.Fatal(err)
	}
	value, err := Get[string](cache, "user", 1)
	if err != nil {
		t.Fatal(err)
	}
	if value != "Ada" {
		t.Fatalf("Get = %q, want Ada", value)
	}
	if _, err := Get[int](cache, "user", 1); err == nil {
		t.Fatal("Get[int] succeeded, want type mismatch")
	}
}

func TestCacheOffBypassesBackend(t *testing.T) {
	backend := &memoryBackend{}
	cache, err := New(Options{Backend: backend})
	if err != nil {
		t.Fatal(err)
	}
	cache.Off()
	if cache.Enabled() {
		t.Fatal("Enabled = true after Off")
	}
	if err := cache.Set("Ada", "user"); err != nil {
		t.Fatal(err)
	}
	if _, err := cache.Get("user"); !errors.Is(err, ErrMiss) {
		t.Fatalf("Get error = %v, want ErrMiss", err)
	}
	if backend.stats != (Stats{}) {
		t.Fatalf("backend stats = %#v, want zero", backend.stats)
	}

	cache.On()
	if !cache.Enabled() {
		t.Fatal("Enabled = false after On")
	}
	if err := cache.Set("Ada", "user"); err != nil {
		t.Fatal(err)
	}
	if _, err := cache.Get("user"); err != nil {
		t.Fatal(err)
	}
	if cache.Stats().Hits != 1 {
		t.Fatalf("Hits = %d, want 1", cache.Stats().Hits)
	}
	if got := cache.Keys(); len(got) != 1 || got[0] != "user" {
		t.Fatalf("Keys = %#v, want [user]", got)
	}
}

func TestCachePublishesDevelopmentEvents(t *testing.T) {
	cache, err := New(Options{Backend: &memoryBackend{}})
	if err != nil {
		t.Fatal(err)
	}
	events, unsubscribe := cache.Subscribe()
	defer unsubscribe()

	if err := cache.Set("Ada", "user", 1); err != nil {
		t.Fatal(err)
	}
	assertCacheEvent(t, events, EventSet, "user-1")

	if _, err := cache.Get("user", 1); err != nil {
		t.Fatal(err)
	}
	assertCacheEvent(t, events, EventHit, "user-1")

	if _, err := cache.Get("missing"); !errors.Is(err, ErrMiss) {
		t.Fatalf("Get missing = %v, want ErrMiss", err)
	}
	miss := assertCacheEvent(t, events, EventMiss, "missing")
	if miss.Stats.Misses != 1 {
		t.Fatalf("miss stats = %#v, want misses=1", miss.Stats)
	}

	cache.Off()
	off := assertCacheEvent(t, events, EventOff, "")
	if off.Enabled {
		t.Fatalf("off event Enabled = true, want false")
	}

	cache.On()
	on := assertCacheEvent(t, events, EventOn, "")
	if !on.Enabled {
		t.Fatalf("on event Enabled = false, want true")
	}
}

func TestCacheContext(t *testing.T) {
	cache, err := New(Options{Backend: &memoryBackend{}})
	if err != nil {
		t.Fatal(err)
	}
	ctx := WithCache(context.Background(), cache)
	got, ok := FromContext(ctx)
	if !ok || got != cache {
		t.Fatalf("FromContext = %#v, %v; want cache, true", got, ok)
	}
	ctx = WithBuildVersion(ctx, "v1.2.3")
	if got, want := BuildVersionFromContext(ctx), "v1.2.3"; got != want {
		t.Fatalf("BuildVersionFromContext = %q, want %q", got, want)
	}
	if got, want := BuildVersionFromContext(context.Background()), "devel"; got != want {
		t.Fatalf("default BuildVersionFromContext = %q, want %q", got, want)
	}
}

func assertCacheEvent(t *testing.T, events <-chan Event, kind EventKind, key string) Event {
	t.Helper()
	select {
	case event := <-events:
		if event.Kind != kind || event.Key != key {
			t.Fatalf("cache event = %#v, want kind=%s key=%q", event, kind, key)
		}
		if event.At.IsZero() {
			t.Fatalf("cache event At is zero: %#v", event)
		}
		return event
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for cache event kind=%s key=%q", kind, key)
		return Event{}
	}
}

func TestWritePrometheus(t *testing.T) {
	cache, err := New(Options{Backend: &memoryBackend{}})
	if err != nil {
		t.Fatal(err)
	}
	if err := cache.Set("Ada", "user"); err != nil {
		t.Fatal(err)
	}
	if _, err := cache.Get("user"); err != nil {
		t.Fatal(err)
	}
	if _, err := cache.Get("missing"); !errors.Is(err, ErrMiss) {
		t.Fatalf("missing get err = %v, want ErrMiss", err)
	}

	var out bytes.Buffer
	if err := WritePrometheus(&out, cache); err != nil {
		t.Fatal(err)
	}
	body := out.String()
	for _, want := range []string{
		"# TYPE golazy_cache_enabled gauge\n",
		"golazy_cache_enabled 1\n",
		"golazy_cache_entries 1\n",
		"golazy_cache_hits_total 1\n",
		"golazy_cache_misses_total 1\n",
		"golazy_cache_sets_total 1\n",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("Prometheus output missing %q in:\n%s", want, body)
		}
	}
}
