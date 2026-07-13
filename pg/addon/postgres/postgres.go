package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"golazy.dev/lazyaddon"
	"golazy.dev/lazyapp"
	"golazy.dev/lazydeps"
	"golazy.dev/lazymigrate"
	"golazy.dev/pg"
	"golazy.dev/pg/pgmigrate"
)

const (
	// AddonID is the stable manifest and runtime identity of the base
	// PostgreSQL add-on.
	AddonID = "postgres"

	defaultDatabaseName    = "postgres"
	defaultDatabaseURLName = "DATABASE_URL"
	databaseURLVariableKey = "database_url_variable"
	postgresDependencyName = "postgres"
	dependenciesCallbackID = "postgres/dependencies"
	migrationsCallbackID   = "postgres/migrations"
)

var (
	addonDefinition = manifestDefinition()
	// Version is the add-on version shipped by this module release. The embedded
	// package manifest is the single version source updated by release tooling.
	Version           = addonDefinition.Version
	addonRegistration = lazyaddon.MustRegisterDefinition(addonDefinition)
)

var (
	// PoolCapability publishes the application-owned PostgreSQL pool to other
	// selected add-ons without relying on a process-global singleton.
	PoolCapability = lazyaddon.DefineCapability[*pgxpool.Pool](addonRegistration, "golazy.dev/pg/postgres/pool", 1)
	// MigrationBackendCapability publishes the PostgreSQL migration backend
	// after the migrations lifecycle hook has run.
	MigrationBackendCapability = lazyaddon.DefineCapability[lazymigrate.Backend](addonRegistration, "golazy.dev/pg/postgres/migration-backend", 1)
)

func init() {
	lazyaddon.MustOn(addonRegistration, lazyapp.DependenciesHook, lazyaddon.CallbackOptions{ID: dependenciesCallbackID}, initializeDependencies)
	lazyaddon.MustOn(addonRegistration, lazyapp.MigrationsHook, lazyaddon.CallbackOptions{ID: migrationsCallbackID}, initializeMigrations)
}

func manifestDefinition() lazyaddon.Definition {
	definition, ok := pg.AddonDefinition(AddonID)
	if !ok {
		panic("postgres add-on: definition is missing from golazy.dev/pg/lazyaddon.toml")
	}
	return definition
}

func initializeDependencies(event *lazyapp.DependenciesEvent) error {
	if event == nil || event.Dependencies == nil {
		return fmt.Errorf("postgres add-on: dependency scope is required")
	}
	if event.Addons == nil {
		return fmt.Errorf("postgres add-on: add-on scope is required")
	}

	var pool *pgxpool.Pool
	_, err := lazydeps.Service(event.Dependencies, postgresDependencyName, func(ctx context.Context) (context.Context, *pgxpool.Pool, error, context.CancelFunc) {
		opened, err := pg.OpenEnv(ctx, databaseURLVariable(event.Addons))
		if err != nil {
			return ctx, nil, fmt.Errorf("open pool: %w", err), nil
		}
		pool = opened
		return pg.WithPool(ctx, opened), opened, nil, opened.Close
	})
	if err != nil {
		return fmt.Errorf("postgres add-on: initialize dependency: %w", err)
	}
	if err := lazyaddon.Provide(event.Addons, addonRegistration, PoolCapability, pool); err != nil {
		return fmt.Errorf("postgres add-on: publish pool: %w", err)
	}
	return nil
}

func initializeMigrations(event *lazyapp.MigrationsEvent) error {
	if event == nil || event.Databases == nil {
		return fmt.Errorf("postgres add-on: migration databases are required")
	}
	pool, err := lazyaddon.Require(event.Addons, PoolCapability)
	if err != nil {
		return fmt.Errorf("postgres add-on: require pool: %w", err)
	}
	if *event.Databases == nil {
		*event.Databases = lazymigrate.Databases{}
	}
	database := (*event.Databases)[defaultDatabaseName]
	if database.Backend != nil {
		return fmt.Errorf("postgres add-on: migration backend for %q is already configured", defaultDatabaseName)
	}
	backend := pgmigrate.New(pool)
	database.Backend = backend
	(*event.Databases)[defaultDatabaseName] = database
	if err := lazyaddon.Provide(event.Addons, addonRegistration, MigrationBackendCapability, lazymigrate.Backend(backend)); err != nil {
		return fmt.Errorf("postgres add-on: publish migration backend: %w", err)
	}
	return nil
}

func databaseURLVariable(scope *lazyaddon.Scope) string {
	if scope != nil {
		if configured := strings.TrimSpace(scope.Config(AddonID)[databaseURLVariableKey]); configured != "" {
			return configured
		}
	}
	return defaultDatabaseURLName
}
