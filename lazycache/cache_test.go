package lazycache

import (
	"context"
	"errors"
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

func TestNewRequiresBackend(t *testing.T) {
	if _, err := New(Options{}); err == nil {
		t.Fatal("New succeeded, want backend error")
	}
}

func TestKeyBuildsStableJoinedParts(t *testing.T) {
	stamp := time.Date(2026, 6, 25, 12, 30, 0, 42, time.FixedZone("CEST", 2*60*60))
	key, err := Key("post", 42, stamp)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := key, "post-42-2026-06-25T10:30:00.000000042Z"; got != want {
		t.Fatalf("Key = %q, want %q", got, want)
	}

	for _, parts := range [][]any{{}, {""}, {nil}} {
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
	if err := cache.Set("Ada", "user"); err != nil {
		t.Fatal(err)
	}
	if _, err := cache.Get("user"); err != nil {
		t.Fatal(err)
	}
	if cache.Stats().Hits != 1 {
		t.Fatalf("Hits = %d, want 1", cache.Stats().Hits)
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
}
