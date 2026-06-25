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

func TestConcurrentAccess(t *testing.T) {
	backend, err := New(Options{MaxEntries: 50})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for worker := range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range 100 {
				key := fmt.Sprintf("%d-%d", worker, i)
				if err := backend.Set(key, i); err != nil {
					t.Errorf("Set(%q): %v", key, err)
					return
				}
				_, _ = backend.Get(key)
			}
		}()
	}
	wg.Wait()
	if stats := backend.Stats(); stats.Entries > 50 {
		t.Fatalf("Entries = %d, want <= 50", stats.Entries)
	}
}
