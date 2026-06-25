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
	Entries    int
	MaxEntries int
	Hits       uint64
	Misses     uint64
	Sets       uint64
	Evictions  uint64
}

// Backend is the storage boundary used by Cache.
type Backend interface {
	Get(key string) (any, error)
	Set(key string, value any) error
	Stats() Stats
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
