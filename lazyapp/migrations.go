package lazyapp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"

	"golazy.dev/lazycontrolplane"
	"golazy.dev/lazymigrate"
)

type migrationMode string

const (
	migrationModeOff  migrationMode = ""
	migrationModeUp   migrationMode = "up"
	migrationModeAuto migrationMode = "auto"
)

var (
	errMigrationInProgress = errors.New("migrations are running")
	exitAfterMigrate       = os.Exit
)

// MigrationsConfig initializes lazymigrate with the dependency-initialized app
// context.
type MigrationsConfig func(context.Context) (lazymigrate.Databases, error)

// Migrations adapts static lazymigrate databases for Config.Migrations.
func Migrations(databases lazymigrate.Databases) MigrationsConfig {
	return func(context.Context) (lazymigrate.Databases, error) {
		return databases, nil
	}
}

func configuredMigrationMode() (migrationMode, error) {
	value := strings.ToLower(strings.TrimSpace(environment.LazyappMigrate))
	switch value {
	case "", "off":
		return migrationModeOff, nil
	case string(migrationModeUp):
		return migrationModeUp, nil
	case string(migrationModeAuto):
		return migrationModeAuto, nil
	default:
		return migrationModeOff, fmt.Errorf("lazyapp: unsupported LAZYAPP_MIGRATE value %q", environment.LazyappMigrate)
	}
}

func applyConfiguredMigrations(ctx context.Context, databases lazymigrate.Databases) error {
	for _, name := range databases.Names() {
		db, ok := databases.Get(name)
		if !ok || !db.HasSources() {
			continue
		}
		migrator, err := databases.Migrator(name)
		if err != nil {
			return err
		}
		if _, err := migrator.Up(ctx, 0); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	return nil
}

type migrationControlPlane struct {
	server *http.Server
	errs   chan error
}

func startMigrationControlPlane() (*migrationControlPlane, error) {
	addr, ok := controlPlaneListenAddr()
	if !ok {
		return nil, nil
	}
	plane := lazycontrolplane.New(lazycontrolplane.Config{
		Readiness: []lazycontrolplane.ReadinessCheck{{
			Name: "migrations",
			Check: func(context.Context) error {
				return errMigrationInProgress
			},
		}},
	})
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("lazyapp: start migration control plane: %w", err)
	}
	server := &http.Server{
		Addr:    addr,
		Handler: plane.StandaloneHandler(),
	}
	control := &migrationControlPlane{
		server: server,
		errs:   make(chan error, 1),
	}
	go func() {
		control.errs <- server.Serve(listener)
	}()
	return control, nil
}

func (control *migrationControlPlane) Close() error {
	if control == nil || control.server == nil {
		return nil
	}
	closeErr := control.server.Close()
	serveErr := <-control.errs
	if errors.Is(serveErr, http.ErrServerClosed) {
		serveErr = nil
	}
	if closeErr != nil && !errors.Is(closeErr, http.ErrServerClosed) {
		return closeErr
	}
	return serveErr
}
