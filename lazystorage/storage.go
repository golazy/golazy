package lazystorage

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"time"
)

// Storage is the minimum read capability for a named object store.
type Storage interface {
	Open(context.Context, string, ...any) (File, []any, error)
}

// File is an opened object. Callers must close it.
type File interface {
	io.Reader
	io.Closer
	Stat() (Info, error)
}

// Info describes a stored object.
type Info struct {
	Key         string
	ContentType string
	Size        int64
	Checksum    string
	ModifiedAt  time.Time
	Metadata    map[string]any
}

// Writer is implemented by storages that can write objects.
type Writer interface {
	Put(context.Context, string, io.Reader, ...any) (Info, []any, error)
}

// Deleter is implemented by storages that can delete objects.
type Deleter interface {
	Delete(context.Context, string, ...any) ([]any, error)
}

// Lister is implemented by storages that can list object keys.
type Lister interface {
	List(context.Context, string, ...any) (Iterator, []any, error)
}

// URLer is implemented by storages that can expose object URLs.
type URLer interface {
	URL(context.Context, string, ...any) (URL, []any, error)
}

// Watcher is implemented by storages that can watch for object changes.
type Watcher interface {
	Watch(context.Context, string, ...any) (Events, []any, error)
}

// URL describes a resolved object URL.
type URL struct {
	String    string
	Public    bool
	ExpiresAt time.Time
}

// Iterator walks object metadata.
type Iterator interface {
	Next() (Info, error)
	Close() error
}

// Events streams storage change events.
type Events interface {
	Next(context.Context) (Event, error)
	Close() error
}

// Event describes a storage change.
type Event struct {
	Key string
	Op  string
}

// ValidateKey validates an object key using io/fs path rules.
func ValidateKey(key string) error {
	if key == "." || !fs.ValidPath(key) {
		return fmt.Errorf("lazystorage: invalid key %q", key)
	}
	return nil
}
