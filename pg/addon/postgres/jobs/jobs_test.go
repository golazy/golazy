package jobs_test

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"golazy.dev/lazyaddon"
	"golazy.dev/lazyapp"
	"golazy.dev/lazydeps"
	"golazy.dev/lazyjobs"
	"golazy.dev/lazyjobs/inmemoryjobs"
	"golazy.dev/lazymigrate"
	"golazy.dev/pg/addon/postgres"
	postgresjobs "golazy.dev/pg/addon/postgres/jobs"
	"golazy.dev/pg/pgjobs"
	"golazy.dev/pg/pgmigrate"
)

func TestAddonResolvesPostgresAndConfiguresJobsAndMigrations(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://postgres:postgres@127.0.0.1:5432/addon_test?sslmode=disable")
	scope, dependencies := initializedJobsScope(t)
	if got, want := scope.Addons(), []string{postgres.AddonID, postgresjobs.AddonID}; !reflect.DeepEqual(got, want) {
		t.Fatalf("add-ons = %v, want %v", got, want)
	}

	databases := lazymigrate.Databases{}
	catalog := new(lazymigrate.Catalog)
	if err := lazyaddon.Run(scope, lazyapp.MigrationsHook, &lazyapp.MigrationsEvent{
		Context:   dependencies.Context(),
		Databases: &databases,
		Catalog:   catalog,
		Addons:    scope,
	}); err != nil {
		t.Fatal(err)
	}
	database := databases[postgres.AddonID]
	if _, ok := database.Backend.(*pgmigrate.Backend); !ok {
		t.Fatalf("migration backend = %T, want *pgmigrate.Backend", database.Backend)
	}
	database.Sources = catalog.Sources(postgres.AddonID)
	if got, want := len(database.Sources), 1; got != want {
		t.Fatalf("migration sources = %d, want %d", got, want)
	}
	migrations, err := database.Sources[0].LoadMigrations(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got, want := migrationIDs(migrations), []string{
		"lazyjobs-202606280002_create_lazy_jobs",
		"lazyjobs-202607090001_add_schedules_and_queue_limits",
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("migration IDs = %v, want %v", got, want)
	}

	define := func(*lazyjobs.JobRunner) {}
	jobsEvent := lazyapp.JobsEvent{
		Context: dependencies.Context(),
		Config: lazyjobs.Config{
			Define:  define,
			Workers: 4,
		},
		Addons: scope,
	}
	if err := lazyaddon.Run(scope, lazyapp.JobsHook, &jobsEvent); err != nil {
		t.Fatal(err)
	}
	if !jobsEvent.Enabled {
		t.Fatal("jobs add-on did not enable the runner")
	}
	backend, ok := jobsEvent.Config.Backend.(*pgjobs.Backend)
	if !ok {
		t.Fatalf("jobs backend = %T, want *pgjobs.Backend", jobsEvent.Config.Backend)
	}
	if jobsEvent.Config.Workers != 4 || reflect.ValueOf(jobsEvent.Config.Define).Pointer() != reflect.ValueOf(define).Pointer() {
		t.Fatal("jobs add-on did not preserve app-owned runner configuration")
	}
	capabilityBackend, err := lazyaddon.Require(scope, postgresjobs.BackendCapability)
	if err != nil {
		t.Fatal(err)
	}
	if capabilityBackend != backend {
		t.Fatalf("backend capability = %p, want %p", capabilityBackend, backend)
	}
}

func TestAddonRejectsAnExistingJobsBackend(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://postgres:postgres@127.0.0.1:5432/addon_test?sslmode=disable")
	scope, dependencies := initializedJobsScope(t)
	event := lazyapp.JobsEvent{
		Context: dependencies.Context(),
		Config:  lazyjobs.Config{Backend: inmemoryjobs.New()},
		Addons:  scope,
	}
	err := lazyaddon.Run(scope, lazyapp.JobsHook, &event)
	if err == nil || !strings.Contains(err.Error(), "already configured") {
		t.Fatalf("jobs hook error = %v, want existing-backend conflict", err)
	}
}

func initializedJobsScope(t *testing.T) (*lazyaddon.Scope, *lazydeps.Scope) {
	t.Helper()
	scope, err := lazyaddon.Resolve(lazyaddon.Select(postgresjobs.AddonID))
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

func migrationIDs(migrations []lazymigrate.Migration) []string {
	ids := make([]string, 0, len(migrations))
	for _, migration := range migrations {
		ids = append(ids, migration.ID)
	}
	return ids
}
