// Package lazycache provides the cache contract used by GoLazy applications.
//
// The package owns the portable parts of application caching: canonical cache
// keys, the enabled/disabled switch, typed Get and Set helpers, statistics, and
// optional development inspection. It intentionally does not import a concrete
// storage backend. A standalone program creates a Backend, passes it to New,
// and then stores values with Cache.Set or the typed Set helper.
//
// Keys are built from one or more non-empty parts with Key. The resulting
// string is stable across callers, and time.Time parts are normalized to UTC so
// helpers in other packages can share the same key format.
//
// Backend is the minimum storage boundary: Get, Set, and Stats. KeyLister and
// EntryInspector are optional capabilities. Cache uses them only when a backend
// exposes them, mainly so development tools can list keys or inspect entries
// without making every backend support that view.
//
// lazyapp is the usual integration point for a GoLazy app. When an application
// does not configure another backend, lazyapp creates the default
// lazycache/inmemorycache backend, wraps it with this package, stores the cache
// on the application context with WithCache, and wires the view cache helpers
// that read it back with FromContext. The cache package stays backend-agnostic;
// the default backend choice belongs to lazyapp.
//
// The in-memory backend is process-local and supports LRU eviction by entry
// count, approximate cached content bytes, or both. Use
// inmemorycache.Options.MaxSizeBytes when production deployments need a hard
// ceiling on retained cached bodies.
//
// When built with the lazydev tag, RegisterLazyDevHandlers exposes cache state
// on the development control plane. lazyapp calls it as part of its control
// plane setup, so normal applications do not need to register those endpoints
// themselves.
package lazycache
