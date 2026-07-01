package pgmigrate

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"golazy.dev/lazymigrate"
)

func TestBackendRunUpAndDown(t *testing.T) {
	databaseURL := os.Getenv("GOLAZY_PG_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set GOLAZY_PG_DATABASE_URL to run PostgreSQL integration tests")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	_, _ = pool.Exec(ctx, `DROP TABLE IF EXISTS lazy_migrations`)
	_, _ = pool.Exec(ctx, `DROP TABLE IF EXISTS pgmigrate_widgets`)

	backend := New(pool)
	if err := backend.Setup(ctx); err != nil {
		t.Fatal(err)
	}

	migration := lazymigrate.Migration{
		ID: "pgmigrate-202606280001_create_widgets",
		Content: []byte(`
-- +lazy Up
CREATE TABLE pgmigrate_widgets (
	id BIGSERIAL PRIMARY KEY,
	name TEXT NOT NULL
);

-- +lazy Down
DROP TABLE pgmigrate_widgets;
`),
	}

	if err := backend.Run(ctx, lazymigrate.Step{Direction: lazymigrate.DirectionUp, Migration: migration}); err != nil {
		t.Fatal(err)
	}
	if err := backend.Run(ctx, lazymigrate.Step{Direction: lazymigrate.DirectionUp, Migration: migration}); err != nil {
		t.Fatalf("repeat up = %v, want no-op", err)
	}
	changedMigration := migration
	changedMigration.Content = []byte(`
-- +lazy Up
CREATE TABLE pgmigrate_changed_widgets (
	id BIGSERIAL PRIMARY KEY
);

-- +lazy Down
DROP TABLE pgmigrate_changed_widgets;
`)
	err = backend.Run(ctx, lazymigrate.Step{Direction: lazymigrate.DirectionUp, Migration: changedMigration})
	if err == nil || !strings.Contains(err.Error(), "different checksum") {
		t.Fatalf("changed repeat up = %v, want checksum mismatch", err)
	}

	applied, err := backend.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(applied) != 1 || applied[0].ID != migration.ID {
		t.Fatalf("unexpected applied migrations: %#v", applied)
	}

	if err := backend.Run(ctx, lazymigrate.Step{Direction: lazymigrate.DirectionDown, Migration: migration}); err != nil {
		t.Fatal(err)
	}
	if err := backend.Run(ctx, lazymigrate.Step{Direction: lazymigrate.DirectionDown, Migration: migration}); err != nil {
		t.Fatalf("repeat down = %v, want no-op", err)
	}
	applied, err = backend.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(applied) != 0 {
		t.Fatalf("expected no applied migrations, got %#v", applied)
	}
}

func TestBackendConcurrentUpIsNoopAfterFirstApply(t *testing.T) {
	databaseURL := os.Getenv("GOLAZY_PG_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set GOLAZY_PG_DATABASE_URL to run PostgreSQL integration tests")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	_, _ = pool.Exec(ctx, `DROP TABLE IF EXISTS lazy_migrations`)
	_, _ = pool.Exec(ctx, `DROP TABLE IF EXISTS pgmigrate_concurrent_widgets`)

	backend := New(pool)
	if err := backend.Setup(ctx); err != nil {
		t.Fatal(err)
	}

	migration := lazymigrate.Migration{
		ID: "pgmigrate-202606280002_create_concurrent_widgets",
		Content: []byte(`
-- +lazy Up
CREATE TABLE pgmigrate_concurrent_widgets (
	id BIGSERIAL PRIMARY KEY,
	name TEXT NOT NULL
);

-- +lazy Down
DROP TABLE pgmigrate_concurrent_widgets;
`),
	}
	step := lazymigrate.Step{Direction: lazymigrate.DirectionUp, Migration: migration}

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- backend.Run(ctx, step)
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}

	applied, err := backend.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(applied) != 1 || applied[0].ID != migration.ID {
		t.Fatalf("unexpected applied migrations: %#v", applied)
	}
}
