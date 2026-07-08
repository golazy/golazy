package inmemorycache

import (
	"container/list"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"golazy.dev/lazycache"
)

// Algorithm names the eviction algorithm used by the cache.
type Algorithm string

const (
	// LRU evicts the least recently used entry when a configured limit is reached.
	LRU Algorithm = "lru"
)

// Options configures an in-memory cache backend.
type Options struct {
	Algorithm    Algorithm
	MaxEntries   int
	MaxSizeBytes int64
}

type entry struct {
	key            string
	value          any
	sizeBytes      int64
	content        string
	contentType    string
	createdAt      time.Time
	updatedAt      time.Time
	lastAccessedAt time.Time
	hits           uint64
	sets           uint64
}

type Cache struct {
	mu           sync.Mutex
	algorithm    Algorithm
	maxEntries   int
	maxSizeBytes int64
	items        map[string]*list.Element
	order        *list.List
	stats        lazycache.Stats
}

// New creates an in-memory lazycache backend.
func New(options Options) (lazycache.Backend, error) {
	algorithm := options.Algorithm
	if algorithm == "" {
		algorithm = LRU
	}
	if algorithm != LRU {
		return nil, fmt.Errorf("inmemorycache: unsupported algorithm %q", algorithm)
	}
	if options.MaxEntries < 0 {
		return nil, fmt.Errorf("inmemorycache: max entries must not be negative")
	}
	if options.MaxSizeBytes < 0 {
		return nil, fmt.Errorf("inmemorycache: max size bytes must not be negative")
	}
	return &Cache{
		algorithm:    algorithm,
		maxEntries:   options.MaxEntries,
		maxSizeBytes: options.MaxSizeBytes,
		items:        map[string]*list.Element{},
		order:        list.New(),
	}, nil
}

// Get returns a value by key.
func (c *Cache) Get(key string) (any, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	element, ok := c.items[key]
	if !ok {
		c.stats.Misses++
		return nil, lazycache.ErrMiss
	}
	c.order.MoveToFront(element)
	item := element.Value.(entry)
	item.lastAccessedAt = time.Now()
	item.hits++
	element.Value = item
	c.stats.Hits++
	return item.value, nil
}

// Set stores value by key.
func (c *Cache) Set(key string, value any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	content, contentType, sizeBytes := inspectValue(value)
	if element, ok := c.items[key]; ok {
		item := element.Value.(entry)
		c.stats.SizeBytes -= item.sizeBytes
		item.value = value
		item.sizeBytes = sizeBytes
		item.content = content
		item.contentType = contentType
		item.updatedAt = now
		item.sets++
		element.Value = item
		c.order.MoveToFront(element)
		c.stats.Sets++
		c.stats.SizeBytes += sizeBytes
		c.enforceLimits()
		return nil
	}

	element := c.order.PushFront(entry{
		key:            key,
		value:          value,
		sizeBytes:      sizeBytes,
		content:        content,
		contentType:    contentType,
		createdAt:      now,
		updatedAt:      now,
		lastAccessedAt: now,
		sets:           1,
	})
	c.items[key] = element
	c.stats.Sets++
	c.stats.SizeBytes += sizeBytes
	c.stats.Entries = len(c.items)
	c.enforceLimits()
	return nil
}

// Stats returns a point-in-time statistics snapshot.
func (c *Cache) Stats() lazycache.Stats {
	c.mu.Lock()
	defer c.mu.Unlock()

	stats := c.stats
	stats.Entries = len(c.items)
	stats.MaxEntries = c.maxEntries
	stats.MaxSizeBytes = c.maxSizeBytes
	return stats
}

// Keys returns a stable snapshot of stored keys.
func (c *Cache) Keys() []string {
	c.mu.Lock()
	defer c.mu.Unlock()

	keys := make([]string, 0, len(c.items))
	for key := range c.items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// Entries returns a stable snapshot of stored entries.
func (c *Cache) Entries() []lazycache.EntryInfo {
	c.mu.Lock()
	defer c.mu.Unlock()

	entries := make([]lazycache.EntryInfo, 0, len(c.items))
	for _, element := range c.items {
		item := element.Value.(entry)
		entries = append(entries, item.info())
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Key < entries[j].Key
	})
	return entries
}

// Entry returns a development-friendly snapshot of one cached value.
func (c *Cache) Entry(key string) (lazycache.EntryDetail, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	element, ok := c.items[key]
	if !ok {
		return lazycache.EntryDetail{}, lazycache.ErrMiss
	}
	item := element.Value.(entry)
	return lazycache.EntryDetail{
		EntryInfo:   item.info(),
		Content:     item.content,
		ContentType: item.contentType,
	}, nil
}

func (c *Cache) enforceLimits() {
	for c.maxEntries > 0 && len(c.items) > c.maxEntries {
		if !c.evictOldest() {
			return
		}
	}
	for c.maxSizeBytes > 0 && c.stats.SizeBytes > c.maxSizeBytes {
		if !c.evictOldest() {
			return
		}
	}
}

func (c *Cache) evictOldest() bool {
	element := c.order.Back()
	if element == nil {
		return false
	}
	item := element.Value.(entry)
	delete(c.items, item.key)
	c.order.Remove(element)
	c.stats.SizeBytes -= item.sizeBytes
	c.stats.Evictions++
	c.stats.Entries = len(c.items)
	return true
}

func (e entry) info() lazycache.EntryInfo {
	return lazycache.EntryInfo{
		Key:            e.key,
		SizeBytes:      e.sizeBytes,
		CreatedAt:      e.createdAt,
		UpdatedAt:      e.updatedAt,
		LastAccessedAt: e.lastAccessedAt,
		Hits:           e.hits,
		Sets:           e.sets,
	}
}

func inspectValue(value any) (content string, contentType string, sizeBytes int64) {
	switch value := value.(type) {
	case string:
		return value, "text/plain; charset=utf-8", int64(len(value))
	case []byte:
		return string(value), "application/octet-stream", int64(len(value))
	}
	if data, err := json.MarshalIndent(value, "", "  "); err == nil {
		return string(data), "application/json; charset=utf-8", int64(len(data))
	}
	content = fmt.Sprintf("%#v", value)
	return content, "text/plain; charset=utf-8", int64(len(content))
}
