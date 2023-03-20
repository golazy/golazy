package lazyservice

import "golang.org/x/exp/slog"

type Service interface {
	Name() string // Name of the service for example production_db
	Kind() string // "db/postgres"
	Start() error
	Stop() error
}

/*
type DependentService interface {
	Dependencies() []string // A service with dependencies, will only start if all dependencies started without errors.
}
*/

type Warmer interface {
	Warmup() error
}

type Logger interface {
	SetLogger(*slog.Logger)
}

type Shutdowner interface {
	Shutdown() error
}

type Installer interface {
	Install(path string) error
}

type Resumer interface {
	// Resume boots the service with a previous state
	// If there is no previous state, it will call Start()
	Resume(state []byte) error

	// The the process is finishing, it will call Detach
	// If not found, it will call Shutdown
	// If not found, it will call Stop()
	Detach() (state []byte, err error)
}

var services = []Service{}

func Register(s Service) {
	services = append(services, s)
}
