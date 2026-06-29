//go:build lazydev

package lazycache

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"golazy.dev/lazycontrolplane"
)

func TestLazyDevCacheHandlersExposeStatsKeysAndToggles(t *testing.T) {
	cache, err := New(Options{Backend: &memoryBackend{}})
	if err != nil {
		t.Fatal(err)
	}
	if err := cache.Set("Ada", "user", 1); err != nil {
		t.Fatal(err)
	}
	if _, err := cache.Get("user", 1); err != nil {
		t.Fatal(err)
	}
	plane := lazycontrolplane.New(lazycontrolplane.Config{})
	RegisterLazyDevHandlers(plane, cache)

	got := requestLazyDevCache(t, plane, http.MethodGet, LazyDevCachePath)
	if !got.Enabled {
		t.Fatal("Enabled = false, want true")
	}
	if got.Stats.Entries != 1 || got.Stats.Sets != 1 || got.Stats.Hits != 1 {
		t.Fatalf("Stats = %#v, want entries=1 sets=1 hits=1", got.Stats)
	}
	if len(got.Keys) != 1 || got.Keys[0] != "user-1" {
		t.Fatalf("Keys = %#v, want [user-1]", got.Keys)
	}
	if len(got.Entries) != 1 || got.Entries[0].Key != "user-1" || got.Entries[0].SizeBytes != 3 {
		t.Fatalf("Entries = %#v, want inspectable user-1 entry", got.Entries)
	}

	detail := requestLazyDevCacheEntry(t, plane, LazyDevCacheEntryPath+"?key=user-1")
	if detail.Key != "user-1" || detail.Content != "Ada" || detail.ContentType == "" {
		t.Fatalf("Entry = %#v, want user-1 content", detail)
	}

	got = requestLazyDevCache(t, plane, http.MethodPost, LazyDevCacheOffPath)
	if got.Enabled {
		t.Fatal("Enabled = true after off")
	}
	if _, err := cache.Get("user", 1); !errors.Is(err, ErrMiss) {
		t.Fatalf("Get while disabled error = %v, want ErrMiss", err)
	}

	got = requestLazyDevCache(t, plane, http.MethodPost, LazyDevCacheOnPath)
	if !got.Enabled {
		t.Fatal("Enabled = false after on")
	}
	if value, err := Get[string](cache, "user", 1); err != nil || value != "Ada" {
		t.Fatalf("Get after on = %q, %v; want Ada, nil", value, err)
	}
}

func requestLazyDevCache(t *testing.T, handler http.Handler, method string, path string) lazyDevCacheResponse {
	t.Helper()
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(method, path, nil))
	if response.Code != http.StatusOK {
		t.Fatalf("%s %s status = %d, want %d: %s", method, path, response.Code, http.StatusOK, response.Body.String())
	}
	if got := response.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
	if got := response.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want JSON", got)
	}
	var out lazyDevCacheResponse
	if err := json.Unmarshal(response.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode cache response: %v\n%s", err, response.Body.String())
	}
	return out
}

func requestLazyDevCacheEntry(t *testing.T, handler http.Handler, path string) EntryDetail {
	t.Helper()
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
	if response.Code != http.StatusOK {
		t.Fatalf("GET %s status = %d, want %d: %s", path, response.Code, http.StatusOK, response.Body.String())
	}
	var out EntryDetail
	if err := json.Unmarshal(response.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode cache entry response: %v\n%s", err, response.Body.String())
	}
	return out
}
