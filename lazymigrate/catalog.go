package lazymigrate

import (
	"context"
	"fmt"
	"strings"
)

type Catalog struct {
	sources map[string][]Source
}

func (c *Catalog) Add(database string, source Source) error {
	database = strings.TrimSpace(database)
	if database == "" {
		return fmt.Errorf("lazymigrate: database name is required")
	}
	if source == nil {
		return fmt.Errorf("lazymigrate: source is required")
	}
	if c.sources == nil {
		c.sources = map[string][]Source{}
	}
	c.sources[database] = append(c.sources[database], source)
	return nil
}

func (c *Catalog) Sources(database string) []Source {
	if c == nil {
		return nil
	}
	sources := c.sources[strings.TrimSpace(database)]
	return append([]Source(nil), sources...)
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
