//go:build lazydev

package lazycache

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"golazy.dev/lazycontrolplane"
	"golazy.dev/lazysse"
)

const LazyDevCachePath = "/cache"
const LazyDevCacheEntryPath = "/cache/entry"
const LazyDevCacheEventsPath = "/cache/events"
const LazyDevCacheOnPath = "/cache/on"
const LazyDevCacheOffPath = "/cache/off"

type lazyDevCacheResponse struct {
	Enabled bool        `json:"enabled"`
	Stats   Stats       `json:"stats"`
	Keys    []string    `json:"keys"`
	Entries []EntryInfo `json:"entries"`
}

// RegisterLazyDevHandlers registers cache inspection and toggle endpoints.
func RegisterLazyDevHandlers(controlPlane *lazycontrolplane.ControlPlane, cache *Cache) {
	if controlPlane == nil {
		return
	}
	controlPlane.Handle("GET "+LazyDevCachePath, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeLazyDevCacheResponse(w, cache)
	}))
	controlPlane.Handle("GET "+LazyDevCacheEntryPath, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeLazyDevCacheEntryResponse(w, cache, r)
	}))
	controlPlane.Handle("GET "+LazyDevCacheEventsPath, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		streamLazyDevCacheEvents(w, r, cache)
	}))
	controlPlane.Handle("POST "+LazyDevCacheOnPath, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if cache == nil {
			writeLazyDevCacheError(w)
			return
		}
		cache.On()
		writeLazyDevCacheResponse(w, cache)
	}))
	controlPlane.Handle("POST "+LazyDevCacheOffPath, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if cache == nil {
			writeLazyDevCacheError(w)
			return
		}
		cache.Off()
		writeLazyDevCacheResponse(w, cache)
	}))
}

func writeLazyDevCacheResponse(w http.ResponseWriter, cache *Cache) {
	if cache == nil {
		writeLazyDevCacheError(w)
		return
	}
	keys := cache.Keys()
	if keys == nil {
		keys = []string{}
	}
	entries := cache.Entries()
	if entries == nil {
		entries = []EntryInfo{}
	}
	response := lazyDevCacheResponse{
		Enabled: cache.Enabled(),
		Stats:   cache.Stats(),
		Keys:    keys,
		Entries: entries,
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, fmt.Sprintf("cache: %v", err), http.StatusInternalServerError)
	}
}

func streamLazyDevCacheEvents(w http.ResponseWriter, r *http.Request, cache *Cache) {
	if cache == nil {
		writeLazyDevCacheError(w)
		return
	}
	stream, err := lazysse.Start(w, r)
	if err != nil {
		http.Error(w, fmt.Sprintf("cache: %v\n", err), http.StatusInternalServerError)
		return
	}
	defer stream.Close()
	stream.Heartbeat(15 * time.Second)

	events, unsubscribe := cache.Subscribe()
	defer unsubscribe()
	for {
		select {
		case <-stream.Done():
			return
		case event, ok := <-events:
			if !ok {
				return
			}
			if err := stream.JSON("cache", event); err != nil {
				return
			}
		}
	}
}

func writeLazyDevCacheEntryResponse(w http.ResponseWriter, cache *Cache, r *http.Request) {
	if cache == nil {
		writeLazyDevCacheError(w)
		return
	}
	key := r.URL.Query().Get("key")
	entry, err := cache.Entry(key)
	if err != nil {
		if errors.Is(err, ErrMiss) {
			http.Error(w, "cache: entry not found\n", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("cache: %v\n", err), http.StatusBadRequest)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(entry); err != nil {
		http.Error(w, fmt.Sprintf("cache: %v", err), http.StatusInternalServerError)
	}
}

func writeLazyDevCacheError(w http.ResponseWriter) {
	http.Error(w, "cache: cache is missing\n", http.StatusInternalServerError)
}
