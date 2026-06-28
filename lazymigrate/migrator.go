package lazymigrate

import (
	"context"
	"fmt"
	"sort"
)

type Config struct {
	Backend Backend
	Sources []Source
}

type Migrator struct {
	backend Backend
	sources []Source
}

func New(config Config) (*Migrator, error) {
	if config.Backend == nil {
		return nil, fmt.Errorf("lazymigrate: backend is required")
	}
	return &Migrator{
		backend: config.Backend,
		sources: append([]Source(nil), config.Sources...),
	}, nil
}

func (m *Migrator) Setup(ctx context.Context) error {
	if m == nil || m.backend == nil {
		return fmt.Errorf("lazymigrate: migrator is not initialized")
	}
	return m.backend.Setup(ctx)
}

func (m *Migrator) List(ctx context.Context) ([]Status, error) {
	sourceMigrations, backendMigrations, err := m.diffInputs(ctx)
	if err != nil {
		return nil, err
	}
	return listStatuses(sourceMigrations, backendMigrations)
}

func (m *Migrator) PlanUp(ctx context.Context, limit int) (Plan, error) {
	statuses, err := m.List(ctx)
	if err != nil {
		return Plan{}, err
	}
	if err := rejectMissing(statuses); err != nil {
		return Plan{}, err
	}
	var steps []Step
	for _, status := range statuses {
		if status.State != StatePending {
			continue
		}
		steps = append(steps, Step{
			Direction: DirectionUp,
			Migration: cloneMigration(status.Migration),
		})
		if limit > 0 && len(steps) >= limit {
			break
		}
	}
	return Plan{Steps: steps}, nil
}

func (m *Migrator) PlanDown(ctx context.Context, limit int) (Plan, error) {
	statuses, err := m.List(ctx)
	if err != nil {
		return Plan{}, err
	}
	if err := rejectMissing(statuses); err != nil {
		return Plan{}, err
	}
	if limit <= 0 {
		limit = 1
	}
	steps := make([]Step, 0, limit)
	for index := len(statuses) - 1; index >= 0; index-- {
		status := statuses[index]
		if status.State != StateApplied {
			continue
		}
		steps = append(steps, Step{
			Direction: DirectionDown,
			Migration: cloneMigration(status.Migration),
		})
		if len(steps) >= limit {
			break
		}
	}
	return Plan{Steps: steps}, nil
}

func (m *Migrator) PlanRedo(ctx context.Context, limit int) (Plan, error) {
	downPlan, err := m.PlanDown(ctx, limit)
	if err != nil {
		return Plan{}, err
	}
	steps := append([]Step(nil), downPlan.Steps...)
	for index := len(downPlan.Steps) - 1; index >= 0; index-- {
		steps = append(steps, Step{
			Direction: DirectionUp,
			Migration: cloneMigration(downPlan.Steps[index].Migration),
		})
	}
	return Plan{Steps: steps}, nil
}

func (m *Migrator) Up(ctx context.Context, limit int) (Plan, error) {
	plan, err := m.PlanUp(ctx, limit)
	if err != nil {
		return Plan{}, err
	}
	if err := m.Apply(ctx, plan); err != nil {
		return plan, err
	}
	return plan, nil
}

func (m *Migrator) Down(ctx context.Context, limit int) (Plan, error) {
	plan, err := m.PlanDown(ctx, limit)
	if err != nil {
		return Plan{}, err
	}
	if err := m.Apply(ctx, plan); err != nil {
		return plan, err
	}
	return plan, nil
}

func (m *Migrator) Redo(ctx context.Context, limit int) (Plan, error) {
	plan, err := m.PlanRedo(ctx, limit)
	if err != nil {
		return Plan{}, err
	}
	if err := m.Apply(ctx, plan); err != nil {
		return plan, err
	}
	return plan, nil
}

func (m *Migrator) Apply(ctx context.Context, plan Plan) error {
	if m == nil || m.backend == nil {
		return fmt.Errorf("lazymigrate: migrator is not initialized")
	}
	if err := m.backend.Setup(ctx); err != nil {
		return err
	}
	for _, step := range plan.Steps {
		if err := step.validate(); err != nil {
			return err
		}
		if err := m.backend.Run(ctx, Step{
			Direction: step.Direction,
			Migration: cloneMigration(step.Migration),
		}); err != nil {
			return err
		}
	}
	return nil
}

func (m *Migrator) DumpSchema(ctx context.Context) ([]byte, error) {
	if m == nil || m.backend == nil {
		return nil, fmt.Errorf("lazymigrate: migrator is not initialized")
	}
	return m.backend.DumpSchema(ctx)
}

func (m *Migrator) LoadSchema(ctx context.Context, schema []byte) error {
	if m == nil || m.backend == nil {
		return fmt.Errorf("lazymigrate: migrator is not initialized")
	}
	return m.backend.LoadSchema(ctx, append([]byte(nil), schema...))
}

func (m *Migrator) diffInputs(ctx context.Context) ([]Migration, []BackendMigration, error) {
	if m == nil || m.backend == nil {
		return nil, nil, fmt.Errorf("lazymigrate: migrator is not initialized")
	}
	sourceMigrations, err := loadSources(ctx, m.sources)
	if err != nil {
		return nil, nil, err
	}
	backendMigrations, err := m.backend.List(ctx)
	if err != nil {
		return nil, nil, err
	}
	if err := validateBackendMigrations(backendMigrations); err != nil {
		return nil, nil, err
	}
	return sourceMigrations, backendMigrations, nil
}

func listStatuses(sourceMigrations []Migration, backendMigrations []BackendMigration) ([]Status, error) {
	applied := map[string]BackendMigration{}
	for _, migration := range backendMigrations {
		applied[migration.ID] = cloneBackendMigration(migration)
	}

	statuses := make([]Status, 0, len(sourceMigrations)+len(backendMigrations))
	sourceIDs := map[string]bool{}
	for _, migration := range sourceMigrations {
		sourceIDs[migration.ID] = true
		backendMigration, ok := applied[migration.ID]
		state := StatePending
		if ok {
			state = StateApplied
		}
		statuses = append(statuses, Status{
			ID:               migration.ID,
			State:            state,
			Migration:        cloneMigration(migration),
			BackendMigration: cloneBackendMigration(backendMigration),
		})
	}

	var missing []BackendMigration
	for _, migration := range backendMigrations {
		if !sourceIDs[migration.ID] {
			missing = append(missing, cloneBackendMigration(migration))
		}
	}
	sort.Slice(missing, func(i, j int) bool {
		return missing[i].ID < missing[j].ID
	})
	for _, migration := range missing {
		statuses = append(statuses, Status{
			ID:               migration.ID,
			State:            StateMissing,
			BackendMigration: cloneBackendMigration(migration),
		})
	}
	return statuses, nil
}

func validateBackendMigrations(migrations []BackendMigration) error {
	seen := map[string]bool{}
	for _, migration := range migrations {
		if migration.ID == "" {
			return fmt.Errorf("lazymigrate: backend migration id is required")
		}
		if seen[migration.ID] {
			return fmt.Errorf("lazymigrate: backend migration %q is duplicated", migration.ID)
		}
		seen[migration.ID] = true
	}
	return nil
}

func rejectMissing(statuses []Status) error {
	for _, status := range statuses {
		if status.State == StateMissing {
			return fmt.Errorf("lazymigrate: backend migration %q is missing from loaded sources", status.ID)
		}
	}
	return nil
}
