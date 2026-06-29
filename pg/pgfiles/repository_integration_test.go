package pgfiles

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"golazy.dev/lazyfiles"
)

func TestMigrationsLoad(t *testing.T) {
	migrations, err := Migrations().LoadMigrations(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(migrations) != 1 || migrations[0].ID != "lazyfiles-202606290001_create_lazy_files" {
		t.Fatalf("unexpected migrations: %#v", migrations)
	}
}

func TestRepositoryPutFindAndDelete(t *testing.T) {
	ctx := context.Background()
	pool := openPool(t, ctx)
	resetSchema(t, ctx, pool)

	repo := New(pool)
	metadata := json.RawMessage(`{"owner":"uploads"}`)
	file, _, err := repo.Put(ctx, lazyfiles.File{
		ID:          "file-1",
		Filename:    "card.txt",
		ContentType: "text/plain",
		Size:        4,
		Checksum:    "sha256:abcd",
		Metadata:    metadata,
	}, lazyfiles.Location{
		Storage:  "local",
		Key:      "card.txt",
		Checksum: "sha256:abcd",
	}, "kept")
	if err != nil {
		t.Fatal(err)
	}
	if file.CreatedAt.IsZero() || file.UpdatedAt.IsZero() {
		t.Fatalf("timestamps were not set: %#v", file)
	}

	got, locations, remaining, err := repo.Find(ctx, lazyfiles.Query{ID: "file-1"}, "kept")
	if err != nil {
		t.Fatal(err)
	}
	if len(remaining) != 1 || remaining[0] != "kept" {
		t.Fatalf("remaining options = %#v, want kept", remaining)
	}
	if got.Filename != "card.txt" || !sameJSON(t, got.Metadata, metadata) {
		t.Fatalf("file = %#v, metadata %s", got, got.Metadata)
	}
	if len(locations) != 1 {
		t.Fatalf("locations = %#v, want one", locations)
	}
	if locations[0].Role != lazyfiles.RolePrimary || locations[0].Status != lazyfiles.StatusActive {
		t.Fatalf("location defaults = %#v", locations[0])
	}

	if _, err := repo.Delete(ctx, "file-1"); err != nil {
		t.Fatal(err)
	}
	_, _, _, err = repo.Find(ctx, lazyfiles.Query{ID: "file-1"})
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Find deleted error = %v, want os.ErrNotExist", err)
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
	_, _ = pool.Exec(ctx, `DROP TABLE IF EXISTS lazy_file_locations`)
	_, _ = pool.Exec(ctx, `DROP TABLE IF EXISTS lazy_files`)
	if _, err := pool.Exec(ctx, `
CREATE TABLE lazy_files (
    id TEXT PRIMARY KEY,
    filename TEXT NOT NULL DEFAULT '',
    content_type TEXT NOT NULL DEFAULT '',
    size BIGINT NOT NULL DEFAULT 0,
    checksum TEXT NOT NULL DEFAULT '',
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE lazy_file_locations (
    file_id TEXT NOT NULL REFERENCES lazy_files(id) ON DELETE CASCADE,
    storage TEXT NOT NULL,
    key TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'primary',
    status TEXT NOT NULL DEFAULT 'active',
    checksum TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (file_id, storage, key)
);
`); err != nil {
		t.Fatal(err)
	}
}
