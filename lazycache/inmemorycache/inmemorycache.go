package inmemorycache

import (
	"container/list"
	"fmt"
	"sync"

	"golazy.dev/lazycache"
)

// Algorithm names the eviction algorithm used by the cache.
type Algorithm string

const (
	// LRU evicts the least recently used entry when MaxEntries is reached.
	LRU Algorithm = "lru"
)

// Options configures an in-memory cache backend.
type Options struct {
	Algorithm  Algorithm
	MaxEntries int
}

type entry struct {
	key   string
	value any
}

type Cache struct {
	mu         sync.Mutex
	algorithm  Algorithm
	maxEntries int
	items      map[string]*list.Element
	order      *list.List
	stats      lazycache.Stats
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
	return &Cache{
		algorithm:  algorithm,
		maxEntries: options.MaxEntries,
		items:      map[string]*list.Element{},
		order:      list.New(),
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
	c.stats.Hits++
	return element.Value.(entry).value, nil
}

// Set stores value by key.
func (c *Cache) Set(key string, value any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if element, ok := c.items[key]; ok {
		element.Value = entry{key: key, value: value}
		c.order.MoveToFront(element)
		c.stats.Sets++
		return nil
	}

	element := c.order.PushFront(entry{key: key, value: value})
	c.items[key] = element
	c.stats.Sets++
	c.stats.Entries = len(c.items)
	c.enforceMaxEntries()
	return nil
}

// Stats returns a point-in-time statistics snapshot.
func (c *Cache) Stats() lazycache.Stats {
	c.mu.Lock()
	defer c.mu.Unlock()

	stats := c.stats
	stats.Entries = len(c.items)
	stats.MaxEntries = c.maxEntries
	return stats
}

func (c *Cache) enforceMaxEntries() {
	if c.maxEntries == 0 {
		return
	}
	for len(c.items) > c.maxEntries {
		element := c.order.Back()
		if element == nil {
			return
		}
		item := element.Value.(entry)
		delete(c.items, item.key)
		c.order.Remove(element)
		c.stats.Evictions++
		c.stats.Entries = len(c.items)
	}
}
