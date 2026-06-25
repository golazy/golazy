// Package inmemorycache provides the default in-process lazycache backend.
//
// lazyapp uses this package when an application does not configure another
// cache backend. Direct use is appropriate for tests, single-process apps, or
// custom lazycache setup that wants the same LRU implementation.
package inmemorycache
