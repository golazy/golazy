package lazymigrate

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
)

func TestCatalogMountsIndependentMigrationTrees(t *testing.T) {
	var catalog Catalog
	if err := catalog.Mount("postgres", "application", fstest.MapFS{
		"202607130001_create_posts.sql": {Data: []byte("app")},
	}); err != nil {
		t.Fatal(err)
	}
	if err := catalog.Mount("postgres", "postgres/jobs", fstest.MapFS{
		"202607130002_create_jobs.sql": {Data: []byte("jobs")},
	}); err != nil {
		t.Fatal(err)
	}

	migrations, err := catalog.LoadMigrations(context.Background(), "postgres")
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, migration := range migrations {
		paths = append(paths, migration.Path)
	}
	if want := []string{
		"application/202607130001_create_posts.sql",
		"postgres/jobs/202607130002_create_jobs.sql",
	}; !reflect.DeepEqual(paths, want) {
		t.Fatalf("migration paths = %v, want %v", paths, want)
	}
}

func TestCatalogMountRejectsDuplicateNamespace(t *testing.T) {
	var catalog Catalog
	files := fstest.MapFS{"202607130001_one.sql": {Data: []byte("one")}}
	if err := catalog.Mount("postgres", "addon", files); err != nil {
		t.Fatal(err)
	}
	err := catalog.Mount("postgres", "addon", files)
	if err == nil || !strings.Contains(err.Error(), "already registered") {
		t.Fatalf("second Mount error = %v, want duplicate namespace", err)
	}
}
