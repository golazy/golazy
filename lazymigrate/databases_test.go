package lazymigrate_test

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"

	"golazy.dev/lazymigrate"
	"golazy.dev/lazymigrate/fakemigrator"
)

func TestDBMigratorCombinesFilesAndSources(t *testing.T) {
	files := fstest.MapFS{
		"migrations/postgres/202603010000_app.sql": {
			Data: []byte("app"),
		},
	}
	db := lazymigrate.DB{
		Backend: fakemigrator.New(),
		Files:   files,
		Sources: []lazymigrate.Source{source("202603020000_package")},
	}

	migrator, err := db.Migrator("postgres")
	if err != nil {
		t.Fatalf("Migrator() error = %v", err)
	}
	plan, err := migrator.PlanUp(context.Background(), 0)
	if err != nil {
		t.Fatalf("PlanUp() error = %v", err)
	}
	got := make([]string, 0, len(plan.Steps))
	for _, step := range plan.Steps {
		got = append(got, step.Migration.ID)
	}
	want := []string{"202603010000_app", "202603020000_package"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("steps = %v, want %v", got, want)
	}
}

func TestDatabasesNamesAndMigrator(t *testing.T) {
	postgres := fakemigrator.New()
	sqlite := fakemigrator.New()
	databases := lazymigrate.Databases{
		"sqlite":     {Backend: sqlite},
		" postgres ": {Backend: postgres},
		"":           {Backend: fakemigrator.New()},
	}

	if got, want := databases.Names(), []string{"postgres", "sqlite"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Names() = %v, want %v", got, want)
	}

	migrator, err := databases.Migrator("postgres")
	if err != nil {
		t.Fatalf("Migrator() error = %v", err)
	}
	if _, err := migrator.Up(context.Background(), 0); err != nil {
		t.Fatalf("Up() error = %v", err)
	}
	if postgres.SetupCount() != 1 {
		t.Fatalf("postgres setup count = %d, want 1", postgres.SetupCount())
	}
	if sqlite.SetupCount() != 0 {
		t.Fatalf("sqlite setup count = %d, want 0", sqlite.SetupCount())
	}
}

func TestDatabasesMigratorErrors(t *testing.T) {
	_, err := (lazymigrate.Databases{}).Migrator("postgres")
	if err == nil || !strings.Contains(err.Error(), `database "postgres" is not configured`) {
		t.Fatalf("missing database error = %v", err)
	}

	_, err = (lazymigrate.Databases{
		"postgres": {},
	}).Migrator("postgres")
	if err == nil || !strings.Contains(err.Error(), "backend is required") {
		t.Fatalf("missing backend error = %v, want backend error", err)
	}
}
