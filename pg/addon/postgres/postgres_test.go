package postgres_test

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"golazy.dev/lazyaddon"
	"golazy.dev/lazyapp"
	"golazy.dev/lazydeps"
	"golazy.dev/lazymigrate"
	"golazy.dev/pg"
	"golazy.dev/pg/addon/postgres"
	"golazy.dev/pg/pgmigrate"
)

func TestAddonInitializesPoolAndMigrationBackend(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("GOLAZY_ADDON_TEST_DATABASE_URL", "postgres://postgres:postgres@127.0.0.1:5432/addon_test?sslmode=disable")

	scope, err := lazyaddon.Resolve(lazyaddon.Selection{Addons: []lazyaddon.Use{{
		ID: postgres.AddonID,
		Config: map[string]string{
			"database_url_variable": "GOLAZY_ADDON_TEST_DATABASE_URL",
		},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := scope.Addons(), []string{postgres.AddonID}; !reflect.DeepEqual(got, want) {
		t.Fatalf("add-ons = %v, want %v", got, want)
	}

	dependencies := lazydeps.New(context.Background())
	t.Cleanup(func() {
		if err := dependencies.Shutdown(context.Background(), "test finished"); err != nil {
			t.Errorf("shutdown dependencies: %v", err)
		}
	})
	if err := lazyaddon.Run(scope, lazyapp.DependenciesHook, &lazyapp.DependenciesEvent{
		Context:      context.Background(),
		Dependencies: dependencies,
		Addons:       scope,
	}); err != nil {
		t.Fatal(err)
	}

	pool, err := lazyaddon.Require(scope, postgres.PoolCapability)
	if err != nil {
		t.Fatal(err)
	}
	if pool == nil {
		t.Fatal("pool capability is nil")
	}
	contextPool, ok := pg.FromContext(dependencies.Context())
	if !ok || contextPool != pool {
		t.Fatalf("dependency context pool = %p, %v; want %p, true", contextPool, ok, pool)
	}

	keepSource := lazymigrate.SourceFunc(func(context.Context) ([]lazymigrate.Migration, error) { return nil, nil })
	databases := lazymigrate.Databases{
		"postgres": {Sources: []lazymigrate.Source{keepSource}},
	}
	if err := lazyaddon.Run(scope, lazyapp.MigrationsHook, &lazyapp.MigrationsEvent{
		Context:   dependencies.Context(),
		Databases: &databases,
		Addons:    scope,
	}); err != nil {
		t.Fatal(err)
	}
	database := databases["postgres"]
	if _, ok := database.Backend.(*pgmigrate.Backend); !ok {
		t.Fatalf("migration backend = %T, want *pgmigrate.Backend", database.Backend)
	}
	if len(database.Sources) != 1 {
		t.Fatalf("migration sources = %d, want existing source preserved", len(database.Sources))
	}
	capabilityBackend, err := lazyaddon.Require(scope, postgres.MigrationBackendCapability)
	if err != nil {
		t.Fatal(err)
	}
	if capabilityBackend != database.Backend {
		t.Fatalf("migration backend capability = %p, want %p", capabilityBackend, database.Backend)
	}
}

func TestAddonRejectsAnExistingMigrationBackend(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://postgres:postgres@127.0.0.1:5432/addon_test?sslmode=disable")
	scope, dependencies := initializedScope(t)
	databases := lazymigrate.Databases{
		"postgres": {Backend: pgmigrate.New(mustPool(t, scope))},
	}
	err := lazyaddon.Run(scope, lazyapp.MigrationsHook, &lazyapp.MigrationsEvent{
		Context:   dependencies.Context(),
		Databases: &databases,
		Addons:    scope,
	})
	if err == nil || !strings.Contains(err.Error(), "already configured") {
		t.Fatalf("migrations hook error = %v, want existing-backend conflict", err)
	}
}

func TestPackageManifestMatchesRuntimeDefinitions(t *testing.T) {
	base, ok := pg.AddonDefinition(postgres.AddonID)
	if !ok {
		t.Fatalf("manifest definition %q is missing", postgres.AddonID)
	}
	if got, want := base.Version, postgres.Version; got != want {
		t.Fatalf("base version = %q, want %q", got, want)
	}
	jobs, ok := pg.AddonDefinition("postgres/jobs")
	if !ok {
		t.Fatal("manifest definition \"postgres/jobs\" is missing")
	}
	if got, want := jobs.Requires, []string{postgres.AddonID + "@" + postgres.Version}; !reflect.DeepEqual(got, want) {
		t.Fatalf("jobs requirements = %v, want %v", got, want)
	}
}

func initializedScope(t *testing.T) (*lazyaddon.Scope, *lazydeps.Scope) {
	t.Helper()
	scope, err := lazyaddon.Resolve(lazyaddon.Select(postgres.AddonID))
	if err != nil {
		t.Fatal(err)
	}
	dependencies := lazydeps.New(context.Background())
	t.Cleanup(func() {
		if err := dependencies.Shutdown(context.Background(), "test finished"); err != nil {
			t.Errorf("shutdown dependencies: %v", err)
		}
	})
	if err := lazyaddon.Run(scope, lazyapp.DependenciesHook, &lazyapp.DependenciesEvent{
		Context:      context.Background(),
		Dependencies: dependencies,
		Addons:       scope,
	}); err != nil {
		t.Fatal(err)
	}
	return scope, dependencies
}

func mustPool(t *testing.T, scope *lazyaddon.Scope) *pgxpool.Pool {
	t.Helper()
	pool, err := lazyaddon.Require(scope, postgres.PoolCapability)
	if err != nil {
		t.Fatal(err)
	}
	return pool
}
