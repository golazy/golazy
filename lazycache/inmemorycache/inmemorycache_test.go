package inmemorycache

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"

	"golazy.dev/lazycache"
)

func TestNewRejectsUnsupportedAlgorithm(t *testing.T) {
	if _, err := New(Options{Algorithm: "fifo"}); err == nil {
		t.Fatal("New succeeded, want unsupported algorithm error")
	}
	if _, err := New(Options{MaxEntries: -1}); err == nil {
		t.Fatal("New succeeded, want max entries error")
	}
	if _, err := New(Options{MaxSizeBytes: -1}); err == nil {
		t.Fatal("New succeeded, want max size bytes error")
	}
}

func TestLRUEvictsLeastRecentlyUsed(t *testing.T) {
	backend, err := New(Options{MaxEntries: 2})
	if err != nil {
		t.Fatal(err)
	}
	if err := backend.Set("a", 1); err != nil {
		t.Fatal(err)
	}
	if err := backend.Set("b", 2); err != nil {
		t.Fatal(err)
	}
	if _, err := backend.Get("a"); err != nil {
		t.Fatal(err)
	}
	if err := backend.Set("c", 3); err != nil {
		t.Fatal(err)
	}

	if _, err := backend.Get("b"); !errors.Is(err, lazycache.ErrMiss) {
		t.Fatalf("Get b error = %v, want ErrMiss", err)
	}
	if value, err := backend.Get("a"); err != nil || value != 1 {
		t.Fatalf("Get a = %v, %v; want 1, nil", value, err)
	}
	if value, err := backend.Get("c"); err != nil || value != 3 {
		t.Fatalf("Get c = %v, %v; want 3, nil", value, err)
	}
	stats := backend.Stats()
	if stats.Entries != 2 || stats.MaxEntries != 2 || stats.Evictions != 1 {
		t.Fatalf("Stats = %#v, want entries=2 max=2 evictions=1", stats)
	}
}

func TestLRUEvictsLeastRecentlyUsedByMaxSize(t *testing.T) {
	backend, err := New(Options{MaxSizeBytes: 10})
	if err != nil {
		t.Fatal(err)
	}
	if err := backend.Set("a", "aaaa"); err != nil {
		t.Fatal(err)
	}
	if err := backend.Set("b", "bbbb"); err != nil {
		t.Fatal(err)
	}
	if _, err := backend.Get("a"); err != nil {
		t.Fatal(err)
	}
	if err := backend.Set("c", "cccc"); err != nil {
		t.Fatal(err)
	}

	if _, err := backend.Get("b"); !errors.Is(err, lazycache.ErrMiss) {
		t.Fatalf("Get b error = %v, want ErrMiss", err)
	}
	if value, err := backend.Get("a"); err != nil || value != "aaaa" {
		t.Fatalf("Get a = %v, %v; want aaaa, nil", value, err)
	}
	if value, err := backend.Get("c"); err != nil || value != "cccc" {
		t.Fatalf("Get c = %v, %v; want cccc, nil", value, err)
	}
	stats := backend.Stats()
	if stats.Entries != 2 || stats.SizeBytes != 8 || stats.MaxSizeBytes != 10 || stats.Evictions != 1 {
		t.Fatalf("Stats = %#v, want entries=2 size=8 max_size=10 evictions=1", stats)
	}
}

func TestOversizedEntryIsNotRetained(t *testing.T) {
	backend, err := New(Options{MaxSizeBytes: 3})
	if err != nil {
		t.Fatal(err)
	}
	if err := backend.Set("large", "large"); err != nil {
		t.Fatal(err)
	}

	if _, err := backend.Get("large"); !errors.Is(err, lazycache.ErrMiss) {
		t.Fatalf("Get large error = %v, want ErrMiss", err)
	}
	stats := backend.Stats()
	if stats.Entries != 0 || stats.SizeBytes != 0 || stats.MaxSizeBytes != 3 || stats.Evictions != 1 {
		t.Fatalf("Stats = %#v, want empty cache under max size", stats)
	}
}

func TestUnboundedDefaultDoesNotEvict(t *testing.T) {
	backend, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}
	for i := range 10 {
		if err := backend.Set(fmt.Sprintf("key-%d", i), i); err != nil {
			t.Fatal(err)
		}
	}
	stats := backend.Stats()
	if stats.Entries != 10 || stats.Evictions != 0 || stats.MaxEntries != 0 {
		t.Fatalf("Stats = %#v, want unbounded entries", stats)
	}
}

func TestKeysReturnsSortedSnapshot(t *testing.T) {
	backend, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"posts-2", "posts-1", "home"} {
		if err := backend.Set(key, key); err != nil {
			t.Fatal(err)
		}
	}
	lister, ok := backend.(lazycache.KeyLister)
	if !ok {
		t.Fatal("backend does not implement KeyLister")
	}
	want := []string{"home", "posts-1", "posts-2"}
	if got := lister.Keys(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Keys = %#v, want %#v", got, want)
	}
}

func TestEntriesAndEntryExposeInspectableMetadata(t *testing.T) {
	backend, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err := backend.Set("posts-1", "<p>Ada</p>"); err != nil {
		t.Fatal(err)
	}
	if _, err := backend.Get("posts-1"); err != nil {
		t.Fatal(err)
	}

	inspector, ok := backend.(lazycache.EntryInspector)
	if !ok {
		t.Fatal("backend does not implement EntryInspector")
	}
	entries := inspector.Entries()
	if len(entries) != 1 {
		t.Fatalf("Entries = %#v, want one entry", entries)
	}
	if entries[0].Key != "posts-1" || entries[0].SizeBytes != int64(len("<p>Ada</p>")) {
		t.Fatalf("entry info = %#v, want key and size", entries[0])
	}
	if entries[0].CreatedAt.IsZero() || entries[0].UpdatedAt.IsZero() || entries[0].LastAccessedAt.IsZero() {
		t.Fatalf("entry timestamps are not populated: %#v", entries[0])
	}
	if entries[0].Hits != 1 || entries[0].Sets != 1 {
		t.Fatalf("entry counters = hits %d sets %d, want 1/1", entries[0].Hits, entries[0].Sets)
	}
	detail, err := inspector.Entry("posts-1")
	if err != nil {
		t.Fatal(err)
	}
	if detail.Content != "<p>Ada</p>" || detail.ContentType != "text/plain; charset=utf-8" {
		t.Fatalf("Entry = %#v, want text content", detail)
	}
	if detail.Hits != 1 || detail.Sets != 1 {
		t.Fatalf("Entry counters = hits %d sets %d, want 1/1", detail.Hits, detail.Sets)
	}
	if stats := backend.Stats(); stats.SizeBytes != int64(len("<p>Ada</p>")) {
		t.Fatalf("Stats.SizeBytes = %d, want %d", stats.SizeBytes, len("<p>Ada</p>"))
	}
}

func TestConcurrentAccess(t *testing.T) {
	backend, err := New(Options{MaxEntries: 50})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for worker := range 8 {
		wg.Go(func() {
			for i := range 100 {
				key := fmt.Sprintf("%d-%d", worker, i)
				if err := backend.Set(key, i); err != nil {
					t.Errorf("Set(%q): %v", key, err)
					return
				}
				_, _ = backend.Get(key)
			}
		})
	}
	wg.Wait()
	if stats := backend.Stats(); stats.Entries > 50 {
		t.Fatalf("Entries = %d, want <= 50", stats.Entries)
	}
}
