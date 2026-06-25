//go:build lazydev

package lazycache

import (
	"encoding/json"
	"fmt"
	"net/http"

	"golazy.dev/lazycontrolplane"
)

const LazyDevCachePath = "/cache"
const LazyDevCacheOnPath = "/cache/on"
const LazyDevCacheOffPath = "/cache/off"

type lazyDevCacheResponse struct {
	Enabled bool     `json:"enabled"`
	Stats   Stats    `json:"stats"`
	Keys    []string `json:"keys"`
}

// RegisterLazyDevHandlers registers cache inspection and toggle endpoints.
func RegisterLazyDevHandlers(controlPlane *lazycontrolplane.ControlPlane, cache *Cache) {
	if controlPlane == nil {
		return
	}
	controlPlane.Handle("GET "+LazyDevCachePath, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeLazyDevCacheResponse(w, cache)
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
	response := lazyDevCacheResponse{
		Enabled: cache.Enabled(),
		Stats:   cache.Stats(),
		Keys:    keys,
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, fmt.Sprintf("cache: %v", err), http.StatusInternalServerError)
	}
}

func writeLazyDevCacheError(w http.ResponseWriter) {
	http.Error(w, "cache: cache is missing\n", http.StatusInternalServerError)
}
