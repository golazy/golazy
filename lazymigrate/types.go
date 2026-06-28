package lazymigrate

import "fmt"

type Direction string

const (
	DirectionUp   Direction = "up"
	DirectionDown Direction = "down"
)

type State string

const (
	StateApplied State = "applied"
	StatePending State = "pending"
	StateMissing State = "missing"
)

type Migration struct {
	ID        string
	Prefix    string
	Timestamp string
	Path      string
	Content   []byte
}

type BackendMigration struct {
	ID string
}

type Status struct {
	ID               string
	State            State
	Migration        Migration
	BackendMigration BackendMigration
}

type Step struct {
	Direction Direction
	Migration Migration
}

type Plan struct {
	Steps []Step
}

func (p Plan) Empty() bool {
	return len(p.Steps) == 0
}

func (d Direction) valid() bool {
	return d == DirectionUp || d == DirectionDown
}

func (s Step) validate() error {
	if !s.Direction.valid() {
		return fmt.Errorf("lazymigrate: unsupported direction %q", s.Direction)
	}
	if s.Migration.ID == "" {
		return fmt.Errorf("lazymigrate: migration id is required")
	}
	return nil
}

func cloneMigration(migration Migration) Migration {
	migration.Content = append([]byte(nil), migration.Content...)
	return migration
}

func cloneBackendMigration(migration BackendMigration) BackendMigration {
	return BackendMigration{ID: migration.ID}
}
