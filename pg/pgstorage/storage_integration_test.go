package pgstorage

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"golazy.dev/lazystorage"
)

func TestMigrationsLoad(t *testing.T) {
	migrations, err := Migrations().LoadMigrations(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(migrations) != 1 || migrations[0].ID != "lazystorage-202606290003_create_lazy_storage_objects" {
		t.Fatalf("unexpected migrations: %#v", migrations)
	}
}

func TestStoragePutOpenListAndDelete(t *testing.T) {
	ctx := context.Background()
	pool := openPool(t, ctx)
	resetSchema(t, ctx, pool)

	storage := New(pool)
	info, remaining, err := storage.Put(ctx, "assets/app.txt", strings.NewReader("hello"),
		lazystorage.ContentType{Value: "text/plain"},
		lazystorage.CacheControl{Value: "public, max-age=0"},
		"kept",
	)
	if err != nil {
		t.Fatal(err)
	}
	if info.Key != "assets/app.txt" || info.ContentType != "text/plain" || info.Size != 5 || info.Checksum == "" {
		t.Fatalf("info = %#v", info)
	}
	if len(remaining) != 1 || remaining[0] != "kept" {
		t.Fatalf("remaining options = %#v, want kept", remaining)
	}

	opened, _, err := storage.Open(ctx, "assets/app.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()
	data, err := io.ReadAll(opened)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello" {
		t.Fatalf("content = %q, want hello", data)
	}
	stat, err := opened.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if stat.Key != info.Key || stat.Checksum != info.Checksum {
		t.Fatalf("stat = %#v, want %#v", stat, info)
	}

	if _, _, err := storage.Put(ctx, "assets/other.txt", strings.NewReader("other")); err != nil {
		t.Fatal(err)
	}
	iterator, _, err := storage.List(ctx, "assets")
	if err != nil {
		t.Fatal(err)
	}
	defer iterator.Close()
	var keys []string
	for {
		next, err := iterator.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		keys = append(keys, next.Key)
	}
	if strings.Join(keys, ",") != "assets/app.txt,assets/other.txt" {
		t.Fatalf("keys = %#v", keys)
	}

	if _, err := storage.Delete(ctx, "assets/app.txt"); err != nil {
		t.Fatal(err)
	}
	_, _, err = storage.Open(ctx, "assets/app.txt")
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Open deleted error = %v, want os.ErrNotExist", err)
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
	_, _ = pool.Exec(ctx, `DROP TABLE IF EXISTS lazy_storage_objects`)
	if _, err := pool.Exec(ctx, `
CREATE TABLE lazy_storage_objects (
    key TEXT PRIMARY KEY,
    content BYTEA NOT NULL,
    content_type TEXT NOT NULL DEFAULT '',
    size BIGINT NOT NULL DEFAULT 0,
    checksum TEXT NOT NULL DEFAULT '',
    cache_control TEXT NOT NULL DEFAULT '',
    content_disposition TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
`); err != nil {
		t.Fatal(err)
	}
}
