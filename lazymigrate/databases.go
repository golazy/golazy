package lazymigrate

import (
	"fmt"
	"io/fs"
	"sort"
	"strings"
)

// DB describes one logical database's migration backend and sources.
type DB struct {
	Backend Backend
	Files   fs.FS
	Sources []Source
}

// HasSources reports whether db has application files or package sources.
func (db DB) HasSources() bool {
	return db.Files != nil || len(db.Sources) > 0
}

// SourcesFor returns the migration sources for database.
func (db DB) SourcesFor(database string) []Source {
	sources := make([]Source, 0, len(db.Sources)+1)
	if db.Files != nil {
		sources = append(sources, ForDatabase(db.Files, database))
	}
	sources = append(sources, db.Sources...)
	return sources
}

// Migrator returns a migrator for database.
func (db DB) Migrator(database string) (*Migrator, error) {
	database = strings.TrimSpace(database)
	if database == "" {
		return nil, fmt.Errorf("lazymigrate: database name is required")
	}
	return New(Config{
		Backend: db.Backend,
		Sources: db.SourcesFor(database),
	})
}

// Databases maps logical database names to independent migration backends.
type Databases map[string]DB

// Names returns configured database names in stable order.
func (databases Databases) Names() []string {
	names := make([]string, 0, len(databases))
	seen := map[string]bool{}
	for name := range databases {
		name = strings.TrimSpace(name)
		if name != "" && !seen[name] {
			names = append(names, name)
			seen[name] = true
		}
	}
	sort.Strings(names)
	return names
}

// Migrator returns the named database migrator.
func (databases Databases) Migrator(name string) (*Migrator, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("lazymigrate: database name is required")
	}
	db, ok := databases.Get(name)
	if !ok {
		return nil, fmt.Errorf("lazymigrate: database %q is not configured", name)
	}
	return db.Migrator(name)
}

// Get returns the named database configuration.
func (databases Databases) Get(name string) (DB, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return DB{}, false
	}
	if databases == nil {
		return DB{}, false
	}
	db, ok := databases[name]
	if !ok {
		for key, candidate := range databases {
			if strings.TrimSpace(key) == name {
				db = candidate
				ok = true
				break
			}
		}
	}
	if !ok {
		return DB{}, false
	}
	return db, true
}
