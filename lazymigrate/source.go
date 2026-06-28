package lazymigrate

import (
	"context"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strings"
)

const migrationConfigFile = "migrations.toml"

var timestampPattern = regexp.MustCompile(`\d{8,}`)

type Source interface {
	LoadMigrations(context.Context) ([]Migration, error)
}

type SourceFunc func(context.Context) ([]Migration, error)

func (fn SourceFunc) LoadMigrations(ctx context.Context) ([]Migration, error) {
	return fn(ctx)
}

type FS struct {
	Files fs.FS
	Dir   string
}

func (source FS) LoadMigrations(ctx context.Context) ([]Migration, error) {
	if source.Files == nil {
		return nil, fmt.Errorf("lazymigrate: fs source is nil")
	}
	dir := strings.Trim(strings.TrimSpace(source.Dir), "/")
	if dir == "" {
		return nil, fmt.Errorf("lazymigrate: migration directory is required")
	}
	entries, err := fs.ReadDir(source.Files, dir)
	if err != nil {
		return nil, fmt.Errorf("read migrations %s: %w", dir, err)
	}

	migrations := make([]Migration, 0, len(entries))
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if entry.IsDir() || entry.Name() == migrationConfigFile {
			continue
		}
		if path.Ext(entry.Name()) == ".go" {
			return nil, fmt.Errorf("lazymigrate: Go migration files are not supported: %s", entry.Name())
		}
		migrationPath := path.Join(dir, entry.Name())
		content, err := fs.ReadFile(source.Files, migrationPath)
		if err != nil {
			return nil, fmt.Errorf("read migration %s: %w", migrationPath, err)
		}
		migration, err := parseMigrationFile(migrationPath, content)
		if err != nil {
			return nil, err
		}
		migrations = append(migrations, migration)
	}
	return normalizeMigrations(migrations)
}

func ForDatabase(files fs.FS, database string) FS {
	return FS{Files: files, Dir: path.Join("migrations", database)}
}

func parseMigrationFile(migrationPath string, content []byte) (Migration, error) {
	name := path.Base(migrationPath)
	extension := path.Ext(name)
	id := strings.TrimSuffix(name, extension)
	if strings.TrimSpace(id) == "" {
		return Migration{}, fmt.Errorf("lazymigrate: migration id is required for %s", migrationPath)
	}
	match := timestampPattern.FindStringIndex(id)
	if match == nil {
		return Migration{}, fmt.Errorf("lazymigrate: migration %q must include a sortable timestamp", id)
	}
	prefix := strings.Trim(id[:match[0]], "-_ .")
	return Migration{
		ID:        id,
		Prefix:    prefix,
		Timestamp: id[match[0]:match[1]],
		Path:      migrationPath,
		Content:   append([]byte(nil), content...),
	}, nil
}

func normalizeMigrations(migrations []Migration) ([]Migration, error) {
	out := make([]Migration, 0, len(migrations))
	seen := map[string]string{}
	for _, migration := range migrations {
		migration = cloneMigration(migration)
		if migration.ID == "" {
			return nil, fmt.Errorf("lazymigrate: migration id is required")
		}
		if previous, exists := seen[migration.ID]; exists {
			return nil, fmt.Errorf("lazymigrate: migration %q is duplicated in %s and %s", migration.ID, previous, migration.Path)
		}
		seen[migration.ID] = migration.Path
		out = append(out, migration)
	}
	sortMigrations(out)
	return out, nil
}

func sortMigrations(migrations []Migration) {
	sort.Slice(migrations, func(i, j int) bool {
		if migrations[i].Timestamp != migrations[j].Timestamp {
			return migrations[i].Timestamp < migrations[j].Timestamp
		}
		if migrations[i].Prefix != migrations[j].Prefix {
			return migrations[i].Prefix < migrations[j].Prefix
		}
		return migrations[i].ID < migrations[j].ID
	})
}
