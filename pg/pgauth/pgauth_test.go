package pgauth

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"golazy.dev/lazyauth"
)

func TestMigrationsLoad(t *testing.T) {
	migrations, err := Migrations().LoadMigrations(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(migrations) != 1 || migrations[0].ID != "lazyauth-202607060001_create_lazy_auth_users" {
		t.Fatalf("unexpected migrations: %#v", migrations)
	}
}

func TestProviderAuthenticate(t *testing.T) {
	ctx := context.Background()
	pool := openPool(t, ctx)
	resetSchema(t, ctx, pool)

	provider := New(pool)
	user, err := provider.UpsertUser(ctx, UserParams{
		Email:    "Ada@example.com",
		Password: "correct horse battery staple",
		Data: map[string]any{
			"mcps": []string{"petra"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if user.ID != "ada@example.com" || user.Data["email"] != "ada@example.com" {
		t.Fatalf("user = %#v", user)
	}

	authenticated, err := provider.Authenticate(ctx, lazyauth.Credential{
		Kind:       "password",
		Identifier: "ada@example.com",
		Secret:     "correct horse battery staple",
	})
	if err != nil {
		t.Fatal(err)
	}
	if authenticated.ID != user.ID || authenticated.Data["email"] != "ada@example.com" {
		t.Fatalf("authenticated = %#v, want user %q", authenticated, user.ID)
	}

	_, err = provider.Authenticate(ctx, lazyauth.Credential{
		Kind:       "password",
		Identifier: "ada@example.com",
		Secret:     "wrong",
	})
	if !errors.Is(err, lazyauth.ErrInvalidCredentials) {
		t.Fatalf("wrong password error = %v, want ErrInvalidCredentials", err)
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
	_, _ = pool.Exec(ctx, `DROP TABLE IF EXISTS lazy_auth_users`)
	if _, err := pool.Exec(ctx, `
CREATE TABLE lazy_auth_users (
	id TEXT PRIMARY KEY,
	email TEXT NOT NULL,
	password_hash TEXT NOT NULL,
	data JSONB NOT NULL DEFAULT '{}'::jsonb,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX lazy_auth_users_email_lower_idx
	ON lazy_auth_users (lower(email));
`); err != nil {
		t.Fatal(err)
	}
}
