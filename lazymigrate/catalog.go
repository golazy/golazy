package lazymigrate

import (
	"context"
	"fmt"
	"io/fs"
	"strings"
	"sync"

	"golazy.dev/lazyfs"
)

type Catalog struct {
	mu      sync.RWMutex
	sources map[string][]Source
	files   map[string]*lazyfs.FS
}

func (c *Catalog) Add(database string, source Source) error {
	if c == nil {
		return fmt.Errorf("lazymigrate: catalog is nil")
	}
	database = strings.TrimSpace(database)
	if database == "" {
		return fmt.Errorf("lazymigrate: database name is required")
	}
	if source == nil {
		return fmt.Errorf("lazymigrate: source is required")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.sources == nil {
		c.sources = map[string][]Source{}
	}
	c.sources[database] = append(c.sources[database], source)
	return nil
}

// Mount adds a package- or application-owned migration filesystem beneath a
// stable namespace. Mounts for one database share a lazyfs stack, so callers
// may contribute them independently before migrations are loaded. Migration
// IDs still come from file names and remain unique across every namespace.
func (c *Catalog) Mount(database, namespace string, files fs.FS) error {
	if c == nil {
		return fmt.Errorf("lazymigrate: catalog is nil")
	}
	database = strings.TrimSpace(database)
	if database == "" {
		return fmt.Errorf("lazymigrate: database name is required")
	}
	namespace = strings.Trim(strings.TrimSpace(namespace), "/")
	if namespace == "" || !fs.ValidPath(namespace) || namespace == "." {
		return fmt.Errorf("lazymigrate: migration namespace %q is invalid", namespace)
	}
	mounted, err := lazyfs.Mount(namespace, files)
	if err != nil {
		return fmt.Errorf("lazymigrate: mount %q: %w", namespace, err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.files == nil {
		c.files = map[string]*lazyfs.FS{}
	}
	stack := c.files[database]
	if stack == nil {
		stack = lazyfs.New()
		c.files[database] = stack
	}
	if err := stack.Add(mounted, lazyfs.Name(namespace), lazyfs.Owner(namespace)); err != nil {
		return fmt.Errorf("lazymigrate: mount %q for %q: %w", namespace, database, err)
	}
	return nil
}

func (c *Catalog) Sources(database string) []Source {
	if c == nil {
		return nil
	}
	database = strings.TrimSpace(database)
	c.mu.RLock()
	sources := append([]Source(nil), c.sources[database]...)
	files := c.files[database]
	c.mu.RUnlock()
	if files != nil && len(files.Layers()) > 0 {
		sources = append(sources, FromTree(files, "."))
	}
	return sources
}

func (c *Catalog) LoadMigrations(ctx context.Context, database string) ([]Migration, error) {
	if c == nil {
		return nil, fmt.Errorf("lazymigrate: catalog is nil")
	}
	return loadSources(ctx, c.Sources(database))
}

func loadSources(ctx context.Context, sources []Source) ([]Migration, error) {
	var migrations []Migration
	for _, source := range sources {
		if source == nil {
			return nil, fmt.Errorf("lazymigrate: source is required")
		}
		loaded, err := source.LoadMigrations(ctx)
		if err != nil {
			return nil, err
		}
		migrations = append(migrations, loaded...)
	}
	return normalizeMigrations(migrations)
}
