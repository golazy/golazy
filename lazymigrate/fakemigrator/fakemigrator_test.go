package fakemigrator_test

import (
	"context"
	"reflect"
	"testing"

	"golazy.dev/lazymigrate"
	"golazy.dev/lazymigrate/fakemigrator"
)

func TestBackendSetupListRunAndSchema(t *testing.T) {
	ctx := context.Background()
	backend := fakemigrator.New("20260301_app")

	if err := backend.Setup(ctx); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
	if err := backend.Setup(ctx); err != nil {
		t.Fatalf("Setup() second error = %v", err)
	}
	if backend.SetupCount() != 2 {
		t.Fatalf("SetupCount() = %d, want 2", backend.SetupCount())
	}

	if err := backend.Run(ctx, lazymigrate.Step{
		Direction: lazymigrate.DirectionUp,
		Migration: lazymigrate.Migration{
			ID:      "20260302_next",
			Content: []byte("up"),
		},
	}); err != nil {
		t.Fatalf("Run(up) error = %v", err)
	}
	if err := backend.Run(ctx, lazymigrate.Step{
		Direction: lazymigrate.DirectionDown,
		Migration: lazymigrate.Migration{
			ID:      "20260301_app",
			Content: []byte("down"),
		},
	}); err != nil {
		t.Fatalf("Run(down) error = %v", err)
	}

	applied, err := backend.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if got, want := ids(applied), []string{"20260302_next"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ids = %v, want %v", got, want)
	}
	if got, want := runIDs(backend.Runs()), []string{"up:20260302_next", "down:20260301_app"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("runs = %v, want %v", got, want)
	}

	if err := backend.LoadSchema(ctx, []byte("schema")); err != nil {
		t.Fatalf("LoadSchema() error = %v", err)
	}
	schema, err := backend.DumpSchema(ctx)
	if err != nil {
		t.Fatalf("DumpSchema() error = %v", err)
	}
	if string(schema) != "schema" {
		t.Fatalf("schema = %q, want schema", schema)
	}
}

func ids(migrations []lazymigrate.BackendMigration) []string {
	out := make([]string, 0, len(migrations))
	for _, migration := range migrations {
		out = append(out, migration.ID)
	}
	return out
}

func runIDs(runs []fakemigrator.Run) []string {
	out := make([]string, 0, len(runs))
	for _, run := range runs {
		out = append(out, string(run.Step.Direction)+":"+run.Step.Migration.ID)
	}
	return out
}
