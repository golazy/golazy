package fakemigrator

import (
	"context"
	"fmt"
	"sync"

	"golazy.dev/lazymigrate"
)

type Backend struct {
	mu         sync.Mutex
	setupCount int
	applied    []lazymigrate.BackendMigration
	runs       []Run
	schema     []byte
}

type Run struct {
	Step lazymigrate.Step
}

func New(applied ...string) *Backend {
	backend := &Backend{}
	for _, id := range applied {
		backend.applied = append(backend.applied, lazymigrate.BackendMigration{ID: id})
	}
	return backend
}

func (b *Backend) Setup(context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.setupCount++
	return nil
}

func (b *Backend) List(context.Context) ([]lazymigrate.BackendMigration, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return cloneBackendMigrations(b.applied), nil
}

func (b *Backend) Run(_ context.Context, step lazymigrate.Step) error {
	if err := validateStep(step); err != nil {
		return err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.runs = append(b.runs, Run{Step: cloneStep(step)})
	switch step.Direction {
	case lazymigrate.DirectionUp:
		if !contains(b.applied, step.Migration.ID) {
			b.applied = append(b.applied, lazymigrate.BackendMigration{ID: step.Migration.ID})
		}
	case lazymigrate.DirectionDown:
		b.applied = remove(b.applied, step.Migration.ID)
	default:
		return fmt.Errorf("fakemigrator: unsupported direction %q", step.Direction)
	}
	return nil
}

func (b *Backend) DumpSchema(context.Context) ([]byte, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]byte(nil), b.schema...), nil
}

func (b *Backend) LoadSchema(_ context.Context, schema []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.schema = append([]byte(nil), schema...)
	return nil
}

func (b *Backend) SetupCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.setupCount
}

func (b *Backend) Runs() []Run {
	b.mu.Lock()
	defer b.mu.Unlock()
	runs := make([]Run, 0, len(b.runs))
	for _, run := range b.runs {
		runs = append(runs, Run{Step: cloneStep(run.Step)})
	}
	return runs
}

func validateStep(step lazymigrate.Step) error {
	if step.Migration.ID == "" {
		return fmt.Errorf("fakemigrator: migration id is required")
	}
	switch step.Direction {
	case lazymigrate.DirectionUp, lazymigrate.DirectionDown:
		return nil
	default:
		return fmt.Errorf("fakemigrator: unsupported direction %q", step.Direction)
	}
}

func contains(migrations []lazymigrate.BackendMigration, id string) bool {
	for _, migration := range migrations {
		if migration.ID == id {
			return true
		}
	}
	return false
}

func remove(migrations []lazymigrate.BackendMigration, id string) []lazymigrate.BackendMigration {
	out := migrations[:0]
	for _, migration := range migrations {
		if migration.ID != id {
			out = append(out, migration)
		}
	}
	return out
}

func cloneBackendMigrations(migrations []lazymigrate.BackendMigration) []lazymigrate.BackendMigration {
	out := make([]lazymigrate.BackendMigration, 0, len(migrations))
	for _, migration := range migrations {
		out = append(out, lazymigrate.BackendMigration{ID: migration.ID})
	}
	return out
}

func cloneStep(step lazymigrate.Step) lazymigrate.Step {
	step.Migration.Content = append([]byte(nil), step.Migration.Content...)
	return step
}
