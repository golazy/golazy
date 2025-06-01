package lazysupport

// MemCache is a simple in-memory cache.
type MemCache map[any][]byte

// Cache caches the result of fn in the cache.
func (c MemCache) Cache(fn func() ([]byte, error), key ...any) ([]byte, error) {
	out, ok := c[key]
	if ok {
		return out, nil
	}
	out, err := fn()

	if err == nil {
		c[key] = out
	}
	return out, err

}

var DefaultCache = MemCache{}

// Cache caches the result of fn in the cache.
func Cache(fn func() ([]byte, error), key ...any) ([]byte, error) {
	return DefaultCache.Cache(fn, key...)
}
