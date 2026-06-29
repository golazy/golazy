package pgmedia

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"golazy.dev/lazymedia"
)

func TestMigrationsLoad(t *testing.T) {
	migrations, err := Migrations().LoadMigrations(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(migrations) != 1 || migrations[0].ID != "lazymedia-202606290002_create_lazy_media_variants" {
		t.Fatalf("unexpected migrations: %#v", migrations)
	}
}

func TestRepositorySaveFindAndDeleteVariant(t *testing.T) {
	ctx := context.Background()
	pool := openPool(t, ctx)
	resetSchema(t, ctx, pool)

	repo := New(pool)
	spec := json.RawMessage(`{"width":1200}`)
	variant, remaining, err := repo.SaveVariant(ctx, lazymedia.Variant{
		SourceFileID: "source-1",
		VariantKey:   "og",
		Spec:         spec,
		OutputFileID: "output-1",
	}, "kept")
	if err != nil {
		t.Fatal(err)
	}
	if len(remaining) != 1 || remaining[0] != "kept" {
		t.Fatalf("remaining options = %#v, want kept", remaining)
	}
	if variant.Status != lazymedia.StatusReady {
		t.Fatalf("Status = %q, want ready", variant.Status)
	}
	if variant.CreatedAt.IsZero() || variant.UpdatedAt.IsZero() {
		t.Fatalf("timestamps were not set: %#v", variant)
	}

	got, _, err := repo.FindVariant(ctx, "source-1", "og")
	if err != nil {
		t.Fatal(err)
	}
	if got.OutputFileID != "output-1" || !sameJSON(t, got.Spec, spec) {
		t.Fatalf("variant = %#v", got)
	}

	if _, err := repo.DeleteVariant(ctx, "source-1", "og"); err != nil {
		t.Fatal(err)
	}
	_, _, err = repo.FindVariant(ctx, "source-1", "og")
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("FindVariant deleted error = %v, want os.ErrNotExist", err)
	}
}

func openPool(t *testing.T, ctx context.Context) *pgxpool.Pool {
	t.Helper()
	databaseURL := os.Getenv("GOLAZY_PG_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set GOLAZY_PG_DATABASE_URL to run PostgreSQL integration tests")
	}
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func resetSchema(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	_, _ = pool.Exec(ctx, `DROP TABLE IF EXISTS lazy_media_variants`)
	if _, err := pool.Exec(ctx, `
CREATE TABLE lazy_media_variants (
    source_file_id TEXT NOT NULL,
    variant_key TEXT NOT NULL,
    spec JSONB,
    output_file_id TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'ready',
    error TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (source_file_id, variant_key)
);
`); err != nil {
		t.Fatal(err)
	}
}

func sameJSON(t *testing.T, got, want json.RawMessage) bool {
	t.Helper()
	var gotValue any
	if err := json.Unmarshal(got, &gotValue); err != nil {
		t.Fatalf("unmarshal got JSON: %v", err)
	}
	var wantValue any
	if err := json.Unmarshal(want, &wantValue); err != nil {
		t.Fatalf("unmarshal want JSON: %v", err)
	}
	return reflect.DeepEqual(gotValue, wantValue)
}
