package lazyapp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

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
	addr        string
	baseContext *dynamicControlContext
	server      *http.Server
	errs        chan error
}

type migrationReadiness struct {
	running atomic.Bool
}

func newMigrationReadiness() *migrationReadiness {
	readiness := &migrationReadiness{}
	readiness.running.Store(true)
	return readiness
}

func (readiness *migrationReadiness) Done() {
	if readiness != nil {
		readiness.running.Store(false)
	}
}

func (readiness *migrationReadiness) Check(context.Context) error {
	if readiness != nil && readiness.running.Load() {
		return errMigrationInProgress
	}
	return nil
}

func prepareMigrationControlPlane(mode migrationMode, controlPlane *lazycontrolplane.ControlPlane) (*lazycontrolplane.ControlPlane, *migrationReadiness, *migrationControlPlane, error) {
	if mode == migrationModeOff {
		return controlPlane, nil, nil, nil
	}
	addr, ok := controlPlaneListenAddr()
	if !ok {
		return controlPlane, nil, nil, nil
	}

	if controlPlane == nil {
		controlPlane = lazycontrolplane.New(lazycontrolplane.Config{})
	}
	readiness := newMigrationReadiness()
	controlPlane.AddReadinessCheck(lazycontrolplane.ReadinessCheck{
		Name:  "migrations",
		Check: readiness.Check,
	})

	if sameListenAddr(addr, listenAddr()) {
		return controlPlane, readiness, nil, nil
	}

	controlPlane.EnablePprof()
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("lazyapp: start migration control plane: %w", err)
	}
	baseContext := newDynamicControlContext(context.Background())
	server := &http.Server{
		Addr:    addr,
		Handler: controlPlane.StandaloneHandler(),
		BaseContext: func(net.Listener) context.Context {
			return baseContext
		},
	}
	control := &migrationControlPlane{
		addr:        addr,
		baseContext: baseContext,
		server:      server,
		errs:        make(chan error, 1),
	}
	go func() {
		control.errs <- server.Serve(listener)
	}()
	return controlPlane, readiness, control, nil
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

func (control *migrationControlPlane) ActiveOn(addr string) bool {
	if control == nil || control.server == nil {
		return false
	}
	return sameListenAddr(control.addr, addr)
}

func (control *migrationControlPlane) SetBaseContext(ctx context.Context) {
	if control == nil || control.baseContext == nil {
		return
	}
	control.baseContext.Set(ctx)
}

type dynamicControlContext struct {
	current atomic.Value
}

type dynamicControlContextValue struct {
	context context.Context
}

// dynamicControlContext lets the early listener start before app.Context is
// complete while later requests still inherit the final context values.
func newDynamicControlContext(ctx context.Context) *dynamicControlContext {
	base := &dynamicControlContext{}
	base.Set(ctx)
	return base
}

func (ctx *dynamicControlContext) Set(next context.Context) {
	if next == nil {
		next = context.Background()
	}
	ctx.current.Store(dynamicControlContextValue{context: next})
}

func (ctx *dynamicControlContext) Deadline() (time.Time, bool) {
	return time.Time{}, false
}

func (ctx *dynamicControlContext) Done() <-chan struct{} {
	return nil
}

func (ctx *dynamicControlContext) Err() error {
	return nil
}

func (ctx *dynamicControlContext) Value(key any) any {
	if ctx == nil {
		return nil
	}
	current, _ := ctx.current.Load().(dynamicControlContextValue)
	if current.context == nil {
		return nil
	}
	return current.context.Value(key)
}
