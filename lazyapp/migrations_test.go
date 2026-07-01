package lazyapp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"golazy.dev/lazydeps"
	"golazy.dev/lazyjobs"
	"golazy.dev/lazyjobs/inmemoryjobs"
	"golazy.dev/lazymigrate"
	"golazy.dev/lazymigrate/fakemigrator"
)

type appMigrationContextKey struct{}

func TestAppInitializesMigrationsWithDependencyContext(t *testing.T) {
	unsetenv(t, "LAZYAPP_MIGRATE", "CONTROL_PLANE_ADDR")
	reloadEnvironmentForTest(t)
	backend := fakemigrator.New()

	app := New(Config{
		Dependencies: func(deps *lazydeps.Scope) error {
			_, err := lazydeps.Service(deps, "postgres", func(ctx context.Context) (context.Context, string, error, context.CancelFunc) {
				return context.WithValue(ctx, appMigrationContextKey{}, "ready"), "ready", nil, nil
			})
			return err
		},
		Migrations: func(ctx context.Context) (lazymigrate.Databases, error) {
			if got := ctx.Value(appMigrationContextKey{}); got != "ready" {
				return nil, fmt.Errorf("context value = %#v, want ready", got)
			}
			return lazymigrate.Databases{
				"postgres": {
					Backend: backend,
					Sources: []lazymigrate.Source{migrationSource("202603010000_app")},
				},
			}, nil
		},
	})

	if app.Migrations == nil {
		t.Fatal("app.Migrations is nil")
	}
	if got, want := app.Migrations.Names(), []string{"postgres"}; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("migration names = %v, want %v", got, want)
	}
	if runs := backend.Runs(); len(runs) != 0 {
		t.Fatalf("runs = %d, want 0 without LAZYAPP_MIGRATE", len(runs))
	}
}

func TestAppAutoMigrationsRunBeforeJobs(t *testing.T) {
	unsetenv(t, "CONTROL_PLANE_ADDR")
	t.Setenv("LAZYAPP_MIGRATE", "auto")
	reloadEnvironmentForTest(t)
	backend := fakemigrator.New()

	app := New(Config{
		Migrations: Migrations(lazymigrate.Databases{
			"postgres": {
				Backend: backend,
				Sources: []lazymigrate.Source{migrationSource("202603010000_app")},
			},
		}),
		Jobs: func(context.Context) (lazyjobs.Config, error) {
			if runs := backend.Runs(); len(runs) != 1 {
				return lazyjobs.Config{}, fmt.Errorf("migration runs = %d, want 1 before jobs", len(runs))
			}
			return lazyjobs.Config{
				Backend:      inmemoryjobs.New(),
				PollInterval: time.Hour,
			}, nil
		},
	})
	defer app.Jobs.Stop(context.Background())

	if runs := backend.Runs(); len(runs) != 1 {
		t.Fatalf("runs = %d, want 1", len(runs))
	}
}

func TestAppMigrateAutoNoConfigIsNoop(t *testing.T) {
	unsetenv(t, "CONTROL_PLANE_ADDR")
	t.Setenv("LAZYAPP_MIGRATE", "auto")
	reloadEnvironmentForTest(t)

	app := New(Config{})
	if app == nil {
		t.Fatal("app is nil")
	}
}

func TestAppMigrateAutoNoSourcesIsNoop(t *testing.T) {
	unsetenv(t, "CONTROL_PLANE_ADDR")
	t.Setenv("LAZYAPP_MIGRATE", "auto")
	reloadEnvironmentForTest(t)

	app := New(Config{
		Migrations: Migrations(lazymigrate.Databases{
			"postgres": {},
		}),
	})
	if app == nil {
		t.Fatal("app is nil")
	}
}

func TestAppMigrateAutoEmptyPlanIsNoop(t *testing.T) {
	unsetenv(t, "CONTROL_PLANE_ADDR")
	t.Setenv("LAZYAPP_MIGRATE", "auto")
	reloadEnvironmentForTest(t)
	backend := fakemigrator.New("202603010000_app")

	app := New(Config{
		Migrations: Migrations(lazymigrate.Databases{
			"postgres": {
				Backend: backend,
				Sources: []lazymigrate.Source{migrationSource("202603010000_app")},
			},
		}),
	})
	if app == nil {
		t.Fatal("app is nil")
	}
	if runs := backend.Runs(); len(runs) != 0 {
		t.Fatalf("runs = %d, want 0", len(runs))
	}
}

func TestAppMigrateUpExitsAfterNoop(t *testing.T) {
	unsetenv(t, "CONTROL_PLANE_ADDR")
	t.Setenv("LAZYAPP_MIGRATE", "up")
	reloadEnvironmentForTest(t)
	oldExit := exitAfterMigrate
	exitAfterMigrate = func(code int) {
		panic(exitCode(code))
	}
	t.Cleanup(func() {
		exitAfterMigrate = oldExit
	})

	defer func() {
		recovered := recover()
		code, ok := recovered.(exitCode)
		if !ok {
			t.Fatalf("panic = %#v, want exitCode", recovered)
		}
		if code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
	}()
	_ = New(Config{})
}

func TestAppRejectsInvalidMigrateMode(t *testing.T) {
	t.Setenv("LAZYAPP_MIGRATE", "sideways")
	reloadEnvironmentForTest(t)

	defer func() {
		recovered := recover()
		if recovered == nil || !strings.Contains(fmt.Sprint(recovered), "unsupported LAZYAPP_MIGRATE") {
			t.Fatalf("panic = %#v, want invalid migrate mode", recovered)
		}
	}()
	_ = New(Config{})
}

func TestMigrationControlPlaneIsLiveButNotReady(t *testing.T) {
	addr := freeLocalAddr(t)
	t.Setenv("CONTROL_PLANE_ADDR", addr)
	t.Setenv("LAZYAPP_MIGRATE", "auto")
	reloadEnvironmentForTest(t)
	backend := newBlockingMigrationBackend()
	done := make(chan struct{})
	errs := make(chan any, 1)

	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				errs <- recovered
				return
			}
			close(done)
		}()
		_ = New(Config{
			Migrations: Migrations(lazymigrate.Databases{
				"postgres": {
					Backend: backend,
					Sources: []lazymigrate.Source{migrationSource("202603010000_app")},
				},
			}),
		})
	}()

	<-backend.runStarted
	baseURL := "http://" + addr
	liveBody, liveStatus := getBody(t, baseURL+"/livez")
	if liveStatus != http.StatusOK || liveBody != "live\n" {
		t.Fatalf("/livez = %d %q, want 200 live", liveStatus, liveBody)
	}
	readyBody, readyStatus := getBody(t, baseURL+"/readyz")
	if readyStatus != http.StatusServiceUnavailable || !strings.Contains(readyBody, "migrations are running") {
		t.Fatalf("/readyz = %d %q, want migration not ready", readyStatus, readyBody)
	}

	close(backend.releaseRun)
	select {
	case <-done:
	case recovered := <-errs:
		t.Fatalf("New panic = %v", recovered)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for app startup")
	}
}

type exitCode int

type blockingMigrationBackend struct {
	runStarted chan struct{}
	releaseRun chan struct{}
}

func newBlockingMigrationBackend() *blockingMigrationBackend {
	return &blockingMigrationBackend{
		runStarted: make(chan struct{}),
		releaseRun: make(chan struct{}),
	}
}

func (b *blockingMigrationBackend) Setup(context.Context) error { return nil }

func (b *blockingMigrationBackend) List(context.Context) ([]lazymigrate.BackendMigration, error) {
	return nil, nil
}

func (b *blockingMigrationBackend) Run(ctx context.Context, step lazymigrate.Step) error {
	if step.Direction != lazymigrate.DirectionUp {
		return fmt.Errorf("direction = %q, want up", step.Direction)
	}
	close(b.runStarted)
	select {
	case <-b.releaseRun:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (b *blockingMigrationBackend) DumpSchema(context.Context) ([]byte, error) {
	return nil, errors.New("unsupported")
}

func (b *blockingMigrationBackend) LoadSchema(context.Context, []byte) error {
	return errors.New("unsupported")
}

func migrationSource(ids ...string) lazymigrate.Source {
	return lazymigrate.SourceFunc(func(context.Context) ([]lazymigrate.Migration, error) {
		migrations := make([]lazymigrate.Migration, 0, len(ids))
		for _, id := range ids {
			migrations = append(migrations, lazymigrate.Migration{
				ID:        id,
				Timestamp: id[:8],
				Path:      id + ".sql",
				Content:   []byte(id),
			})
		}
		return migrations, nil
	})
}

func freeLocalAddr(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	return addr
}

func getBody(t *testing.T, url string) (string, int) {
	t.Helper()
	response, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(body), response.StatusCode
}
