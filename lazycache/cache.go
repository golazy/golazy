package lazycache

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync/atomic"
	"time"
)

// ErrMiss reports that a key is not available from the cache.
var ErrMiss = errors.New("lazycache: miss")

// Stats is the common cache statistics shape returned by every backend.
type Stats struct {
	Entries    int    `json:"entries"`
	MaxEntries int    `json:"max_entries"`
	SizeBytes  int64  `json:"size_bytes"`
	Hits       uint64 `json:"hits"`
	Misses     uint64 `json:"misses"`
	Sets       uint64 `json:"sets"`
	Evictions  uint64 `json:"evictions"`
}

// Backend is the storage boundary used by Cache.
type Backend interface {
	Get(key string) (any, error)
	Set(key string, value any) error
	Stats() Stats
}

// KeyLister is an optional backend capability used by development tooling.
type KeyLister interface {
	Keys() []string
}

// EntryInfo describes one cached value without exposing its body.
type EntryInfo struct {
	Key            string    `json:"key"`
	SizeBytes      int64     `json:"size_bytes"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	LastAccessedAt time.Time `json:"last_accessed_at"`
}

// EntryDetail describes one cached value with a development-friendly body.
type EntryDetail struct {
	EntryInfo
	Content     string `json:"content"`
	ContentType string `json:"content_type"`
}

// EntryInspector is an optional backend capability used by development tooling.
type EntryInspector interface {
	Entries() []EntryInfo
	Entry(key string) (EntryDetail, error)
}

// Options configures a Cache.
type Options struct {
	Backend Backend
}

// Cache wraps a backend with GoLazy key building and on/off switching.
type Cache struct {
	backend Backend
	enabled atomic.Bool
}

// New creates a cache around a backend.
func New(options Options) (*Cache, error) {
	if options.Backend == nil {
		return nil, fmt.Errorf("lazycache: backend is required")
	}
	cache := &Cache{backend: options.Backend}
	cache.enabled.Store(true)
	return cache, nil
}

// On enables reads and writes.
func (c *Cache) On() {
	if c == nil {
		return
	}
	c.enabled.Store(true)
}

// Off disables reads and turns writes into no-ops.
func (c *Cache) Off() {
	if c == nil {
		return
	}
	c.enabled.Store(false)
}

// Enabled reports whether reads and writes are active.
func (c *Cache) Enabled() bool {
	return c != nil && c.enabled.Load()
}

// Get returns a cached value for the key built from parts.
func (c *Cache) Get(parts ...any) (any, error) {
	if c == nil || c.backend == nil {
		return nil, fmt.Errorf("lazycache: cache is not initialized")
	}
	if !c.enabled.Load() {
		return nil, ErrMiss
	}
	key, err := Key(parts...)
	if err != nil {
		return nil, err
	}
	value, err := c.backend.Get(key)
	if err != nil {
		if errors.Is(err, ErrMiss) {
			return nil, ErrMiss
		}
		return nil, err
	}
	return value, nil
}

// Set stores value under the key built from parts.
func (c *Cache) Set(value any, parts ...any) error {
	if c == nil || c.backend == nil {
		return fmt.Errorf("lazycache: cache is not initialized")
	}
	if !c.enabled.Load() {
		return nil
	}
	key, err := Key(parts...)
	if err != nil {
		return err
	}
	return c.backend.Set(key, value)
}

// Stats returns the backend statistics.
func (c *Cache) Stats() Stats {
	if c == nil || c.backend == nil {
		return Stats{}
	}
	return c.backend.Stats()
}

// Keys returns the backend keys when the backend exposes them.
func (c *Cache) Keys() []string {
	if c == nil || c.backend == nil {
		return nil
	}
	lister, ok := c.backend.(KeyLister)
	if !ok {
		return nil
	}
	return lister.Keys()
}

// Entries returns inspectable backend entries when the backend exposes them.
func (c *Cache) Entries() []EntryInfo {
	if c == nil || c.backend == nil {
		return nil
	}
	inspector, ok := c.backend.(EntryInspector)
	if ok {
		return inspector.Entries()
	}
	keys := c.Keys()
	if len(keys) == 0 {
		return nil
	}
	entries := make([]EntryInfo, 0, len(keys))
	for _, key := range keys {
		entries = append(entries, EntryInfo{Key: key})
	}
	return entries
}

// Entry returns an inspectable backend entry when the backend exposes it.
func (c *Cache) Entry(key string) (EntryDetail, error) {
	if c == nil || c.backend == nil {
		return EntryDetail{}, fmt.Errorf("lazycache: cache is not initialized")
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return EntryDetail{}, fmt.Errorf("lazycache: key part is empty")
	}
	inspector, ok := c.backend.(EntryInspector)
	if !ok {
		return EntryDetail{}, fmt.Errorf("lazycache: backend does not expose cache entries")
	}
	return inspector.Entry(key)
}

// Get returns a cached value with a concrete type.
func Get[T any](cache *Cache, parts ...any) (T, error) {
	var zero T
	value, err := cache.Get(parts...)
	if err != nil {
		return zero, err
	}
	if typed, ok := value.(T); ok {
		return typed, nil
	}

	target := reflect.TypeOf((*T)(nil)).Elem()
	if value == nil {
		if nilAssignableTo(target) {
			return zero, nil
		}
		return zero, fmt.Errorf("lazycache: cached value is nil, not %s", target)
	}
	source := reflect.TypeOf(value)
	if source.AssignableTo(target) {
		return reflect.ValueOf(value).Interface().(T), nil
	}
	return zero, fmt.Errorf("lazycache: cached value has type %s, not %s", source, target)
}

// Set stores a typed value under the key built from parts.
func Set[T any](cache *Cache, value T, parts ...any) error {
	return cache.Set(value, parts...)
}

func nilAssignableTo(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return true
	default:
		return false
	}
}

// Key builds the canonical cache key for a list of parts.
func Key(parts ...any) (string, error) {
	if len(parts) == 0 {
		return "", fmt.Errorf("lazycache: key requires at least one part")
	}
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		value, err := keyPart(part)
		if err != nil {
			return "", err
		}
		out = append(out, value)
	}
	return strings.Join(out, "-"), nil
}

func keyPart(part any) (string, error) {
	if part == nil {
		return "", fmt.Errorf("lazycache: key part is nil")
	}
	if value, ok := part.(time.Time); ok {
		return keyPartString(value.UTC().Format(time.RFC3339Nano))
	}
	if value, ok := part.(*time.Time); ok {
		if value == nil {
			return "", fmt.Errorf("lazycache: key part is nil")
		}
		return keyPartString(value.UTC().Format(time.RFC3339Nano))
	}
	if isNil(part) {
		return "", fmt.Errorf("lazycache: key part is nil")
	}
	return keyPartString(fmt.Sprint(part))
}

func keyPartString(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("lazycache: key part is empty")
	}
	return value, nil
}

func isNil(value any) bool {
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}

type cacheContextKey struct{}
type buildVersionContextKey struct{}

// WithCache returns a context carrying cache.
func WithCache(ctx context.Context, cache *Cache) context.Context {
	return context.WithValue(ctx, cacheContextKey{}, cache)
}

// FromContext returns the cache carried by ctx.
func FromContext(ctx context.Context) (*Cache, bool) {
	if ctx == nil {
		return nil, false
	}
	cache, ok := ctx.Value(cacheContextKey{}).(*Cache)
	return cache, ok && cache != nil
}

// WithBuildVersion returns a context carrying the application build version used by cache keys.
func WithBuildVersion(ctx context.Context, version string) context.Context {
	version = strings.TrimSpace(version)
	if version == "" || version == "(devel)" {
		version = "devel"
	}
	return context.WithValue(ctx, buildVersionContextKey{}, version)
}

// BuildVersionFromContext returns the cache-key build version for ctx.
func BuildVersionFromContext(ctx context.Context) string {
	if ctx == nil {
		return "devel"
	}
	version, _ := ctx.Value(buildVersionContextKey{}).(string)
	version = strings.TrimSpace(version)
	if version == "" || version == "(devel)" {
		return "devel"
	}
	return version
}
