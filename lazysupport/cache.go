package lazysupport

type MemCache map[any][]byte

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

func Cache(fn func() ([]byte, error), key ...any) ([]byte, error) {
	return DefaultCache.Cache(fn, key...)
}
