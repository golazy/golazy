package lazymigrate_test

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"golazy.dev/lazymigrate"
	"golazy.dev/lazymigrate/fakemigrator"
)

func TestMigratorListDiffsSourceAndBackend(t *testing.T) {
	ctx := context.Background()
	migrator := newTestMigrator(t, fakemigrator.New("20260301_app", "20260303_missing"), source("20260301_app", "20260302_next"))

	statuses, err := migrator.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	got := statusPairs(statuses)
	want := []string{
		"20260301_app:applied",
		"20260302_next:pending",
		"20260303_missing:missing",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("statuses = %v, want %v", got, want)
	}
}

func TestMigratorPlansUpDownAndRedo(t *testing.T) {
	ctx := context.Background()
	backend := fakemigrator.New("20260301_app")
	migrator := newTestMigrator(t, backend, source("20260301_app", "20260302_next", "20260303_more"))

	up, err := migrator.PlanUp(ctx, 1)
	if err != nil {
		t.Fatalf("PlanUp() error = %v", err)
	}
	assertSteps(t, up, []string{"up:20260302_next"})

	backend = fakemigrator.New("20260301_app", "20260302_next", "20260303_more")
	migrator = newTestMigrator(t, backend, source("20260301_app", "20260302_next", "20260303_more"))
	down, err := migrator.PlanDown(ctx, 0)
	if err != nil {
		t.Fatalf("PlanDown() error = %v", err)
	}
	assertSteps(t, down, []string{"down:20260303_more"})

	redo, err := migrator.PlanRedo(ctx, 2)
	if err != nil {
		t.Fatalf("PlanRedo() error = %v", err)
	}
	assertSteps(t, redo, []string{
		"down:20260303_more",
		"down:20260302_next",
		"up:20260302_next",
		"up:20260303_more",
	})
}

func TestMigratorMissingSourceBlocksExecutionPlans(t *testing.T) {
	ctx := context.Background()
	migrator := newTestMigrator(t, fakemigrator.New("20260399_missing"), source("20260301_app"))

	_, err := migrator.PlanUp(ctx, 0)
	if err == nil || !strings.Contains(err.Error(), `backend migration "20260399_missing" is missing`) {
		t.Fatalf("PlanUp() error = %v, want missing source error", err)
	}
	_, err = migrator.PlanDown(ctx, 0)
	if err == nil || !strings.Contains(err.Error(), `backend migration "20260399_missing" is missing`) {
		t.Fatalf("PlanDown() error = %v, want missing source error", err)
	}
}

func TestMigratorAppliesPlanThroughBackend(t *testing.T) {
	ctx := context.Background()
	backend := fakemigrator.New()
	migrator := newTestMigrator(t, backend, source("20260301_app", "20260302_next"))

	plan, err := migrator.Up(ctx, 0)
	if err != nil {
		t.Fatalf("Up() error = %v", err)
	}
	assertSteps(t, plan, []string{"up:20260301_app", "up:20260302_next"})
	if backend.SetupCount() != 1 {
		t.Fatalf("SetupCount() = %d, want 1", backend.SetupCount())
	}
	applied, err := backend.List(ctx)
	if err != nil {
		t.Fatalf("backend.List() error = %v", err)
	}
	if got, want := backendIDs(applied), []string{"20260301_app", "20260302_next"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("backend ids = %v, want %v", got, want)
	}
}

func TestMigratorSchemaRoundTrip(t *testing.T) {
	ctx := context.Background()
	migrator := newTestMigrator(t, fakemigrator.New(), source("20260301_app"))

	if err := migrator.LoadSchema(ctx, []byte("schema")); err != nil {
		t.Fatalf("LoadSchema() error = %v", err)
	}
	schema, err := migrator.DumpSchema(ctx)
	if err != nil {
		t.Fatalf("DumpSchema() error = %v", err)
	}
	if string(schema) != "schema" {
		t.Fatalf("schema = %q, want schema", schema)
	}
}

func newTestMigrator(t *testing.T, backend lazymigrate.Backend, sources ...lazymigrate.Source) *lazymigrate.Migrator {
	t.Helper()
	migrator, err := lazymigrate.New(lazymigrate.Config{
		Backend: backend,
		Sources: sources,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return migrator
}

func assertSteps(t *testing.T, plan lazymigrate.Plan, want []string) {
	t.Helper()
	got := make([]string, 0, len(plan.Steps))
	for _, step := range plan.Steps {
		got = append(got, string(step.Direction)+":"+step.Migration.ID)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("steps = %v, want %v", got, want)
	}
}

func statusPairs(statuses []lazymigrate.Status) []string {
	out := make([]string, 0, len(statuses))
	for _, status := range statuses {
		out = append(out, status.ID+":"+string(status.State))
	}
	return out
}

func backendIDs(migrations []lazymigrate.BackendMigration) []string {
	out := make([]string, 0, len(migrations))
	for _, migration := range migrations {
		out = append(out, migration.ID)
	}
	return out
}
