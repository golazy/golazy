package lazymigrate_test

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"

	"golazy.dev/lazymigrate"
)

func TestFromFSSourceLoadsAndSortsMigrations(t *testing.T) {
	files := fstest.MapFS{
		"postgres/migrations.toml": {
			Data: []byte("[postgres]\n"),
		},
		"postgres/nested/202603030000_nested.sql": {
			Data: []byte("nested"),
		},
		"postgres/lazyjobs-20260302.sql": {
			Data: []byte("jobs"),
		},
		"postgres/202603010101_create_documents.sql": {
			Data: []byte("documents"),
		},
		"postgres/lazyassets-20260302.sql": {
			Data: []byte("assets"),
		},
	}

	migrations, err := lazymigrate.FromFS(files, "postgres").LoadMigrations(context.Background())
	if err != nil {
		t.Fatalf("LoadMigrations() error = %v", err)
	}
	got := ids(migrations)
	want := []string{
		"202603010101_create_documents",
		"lazyassets-20260302",
		"lazyjobs-20260302",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ids = %v, want %v", got, want)
	}
	if migrations[1].Prefix != "lazyassets" || migrations[1].Timestamp != "20260302" {
		t.Fatalf("parsed metadata = %#v", migrations[1])
	}
	if migrations[2].Path != "postgres/lazyjobs-20260302.sql" {
		t.Fatalf("path = %q", migrations[2].Path)
	}
}

func TestForDatabaseReadsConventionalDirectory(t *testing.T) {
	files := fstest.MapFS{
		"db/postgres/migrations/202603010101_create_documents.sql": {
			Data: []byte("documents"),
		},
	}

	migrations, err := lazymigrate.ForDatabase(files, "postgres").LoadMigrations(context.Background())
	if err != nil {
		t.Fatalf("LoadMigrations() error = %v", err)
	}
	if got, want := ids(migrations), []string{"202603010101_create_documents"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ids = %v, want %v", got, want)
	}
	if migrations[0].Path != "db/postgres/migrations/202603010101_create_documents.sql" {
		t.Fatalf("path = %q", migrations[0].Path)
	}
}

func TestFSSourceRejectsDuplicateIDs(t *testing.T) {
	files := fstest.MapFS{
		"db/postgres/migrations/20260302.sql": {Data: []byte("sql")},
		"db/postgres/migrations/20260302.txt": {Data: []byte("txt")},
		"db/postgres/migrations/20260303.sql": {Data: []byte("other")},
	}

	_, err := lazymigrate.ForDatabase(files, "postgres").LoadMigrations(context.Background())
	if err == nil || !strings.Contains(err.Error(), `migration "20260302" is duplicated`) {
		t.Fatalf("LoadMigrations() error = %v, want duplicate id", err)
	}
}

func TestFSSourceRejectsMalformedFilenames(t *testing.T) {
	files := fstest.MapFS{
		"db/postgres/migrations/create_documents.sql": {Data: []byte("sql")},
	}

	_, err := lazymigrate.ForDatabase(files, "postgres").LoadMigrations(context.Background())
	if err == nil || !strings.Contains(err.Error(), "must include a sortable timestamp") {
		t.Fatalf("LoadMigrations() error = %v, want timestamp error", err)
	}
}

func TestCatalogCombinesSourcesAndDetectsCollisions(t *testing.T) {
	ctx := context.Background()
	var catalog lazymigrate.Catalog
	if err := catalog.Add("postgres", source("20260301_app")); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if err := catalog.Add("postgres", source("20260302_package")); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	migrations, err := catalog.LoadMigrations(ctx, "postgres")
	if err != nil {
		t.Fatalf("LoadMigrations() error = %v", err)
	}
	if got, want := ids(migrations), []string{"20260301_app", "20260302_package"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ids = %v, want %v", got, want)
	}

	if err := catalog.Add("postgres", source("20260301_app")); err != nil {
		t.Fatalf("Add() duplicate source error = %v", err)
	}
	_, err = catalog.LoadMigrations(ctx, "postgres")
	if err == nil || !strings.Contains(err.Error(), `migration "20260301_app" is duplicated`) {
		t.Fatalf("LoadMigrations() duplicate error = %v", err)
	}
}

func source(ids ...string) lazymigrate.Source {
	return lazymigrate.SourceFunc(func(context.Context) ([]lazymigrate.Migration, error) {
		migrations := make([]lazymigrate.Migration, 0, len(ids))
		for _, id := range ids {
			migrations = append(migrations, lazymigrate.Migration{
				ID:        id,
				Timestamp: id[:8],
				Path:      id + ".sql",
				Content:   []byte(id),
			})
		}
		return migrations, nil
	})
}

func ids(migrations []lazymigrate.Migration) []string {
	out := make([]string, 0, len(migrations))
	for _, migration := range migrations {
		out = append(out, migration.ID)
	}
	return out
}
