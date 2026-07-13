package jobs

import (
	"fmt"

	"golazy.dev/lazyaddon"
	"golazy.dev/lazyapp"
	"golazy.dev/lazyjobs"
	"golazy.dev/pg"
	"golazy.dev/pg/addon/postgres"
	"golazy.dev/pg/pgjobs"
)

const (
	// AddonID is the stable manifest and runtime identity of the PostgreSQL
	// jobs add-on.
	AddonID = "postgres/jobs"

	migrationsCallbackID = "postgres/jobs/migrations"
	jobsCallbackID       = "postgres/jobs/backend"
)

var (
	addonDefinition = manifestDefinition()
	// Version is the add-on version shipped by this module release.
	Version           = addonDefinition.Version
	addonRegistration = lazyaddon.MustRegisterDefinition(addonDefinition)
)

var (
	// BackendCapability publishes the selected application's PostgreSQL jobs
	// backend to other add-ons after the jobs lifecycle hook has run.
	BackendCapability = lazyaddon.DefineCapability[lazyjobs.Backend](addonRegistration, "golazy.dev/pg/postgres/jobs/backend", 1)
)

func init() {
	lazyaddon.MustOn(addonRegistration, lazyapp.MigrationsHook, lazyaddon.CallbackOptions{
		ID:    migrationsCallbackID,
		After: []string{"postgres/migrations"},
	}, addMigrations)
	lazyaddon.MustOn(addonRegistration, lazyapp.JobsHook, lazyaddon.CallbackOptions{ID: jobsCallbackID}, initializeJobs)
}

func manifestDefinition() lazyaddon.Definition {
	definition, ok := pg.AddonDefinition(AddonID)
	if !ok {
		panic("postgres/jobs add-on: definition is missing from golazy.dev/pg/lazyaddon.toml")
	}
	return definition
}

func addMigrations(event *lazyapp.MigrationsEvent) error {
	if event == nil || event.Databases == nil || *event.Databases == nil {
		return fmt.Errorf("postgres/jobs add-on: PostgreSQL migration database is required")
	}
	if event.Catalog == nil {
		return fmt.Errorf("postgres/jobs add-on: migration catalog is required")
	}
	database, exists := (*event.Databases)[postgres.AddonID]
	if !exists || database.Backend == nil {
		return fmt.Errorf("postgres/jobs add-on: PostgreSQL migration backend is required")
	}
	return event.Catalog.Mount(postgres.AddonID, AddonID, pgjobs.MigrationFiles())
}

func initializeJobs(event *lazyapp.JobsEvent) error {
	if event == nil {
		return fmt.Errorf("postgres/jobs add-on: jobs event is required")
	}
	if event.Config.Backend != nil {
		return fmt.Errorf("postgres/jobs add-on: jobs backend is already configured")
	}
	pool, err := lazyaddon.Require(event.Addons, postgres.PoolCapability)
	if err != nil {
		return fmt.Errorf("postgres/jobs add-on: require pool: %w", err)
	}
	backend := pgjobs.New(pool)
	event.Config.Backend = backend
	event.Enabled = true
	if err := lazyaddon.Provide(event.Addons, addonRegistration, BackendCapability, lazyjobs.Backend(backend)); err != nil {
		return fmt.Errorf("postgres/jobs add-on: publish backend: %w", err)
	}
	return nil
}
