package withpg

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"testing"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/jackc/pgx/v5"
)

var startMu sync.Mutex

type Config struct {
	SchemaFile string
	PgVersion  string
	PgVersions []string
	DataPath   string
	DBName     string
	Port       uint32
	Logger     io.Writer
}

type DB struct {
	state *dbState
}

type dbState struct {
	mu     sync.Mutex
	url    string
	conn   *pgx.Conn
	stopFn func() error
}

func (db DB) URL() string {
	if db.state == nil {
		return ""
	}
	return db.state.url
}

func (db DB) Client(ctx context.Context) (*pgx.Conn, error) {
	if db.state == nil {
		return nil, fmt.Errorf("withpg: database is not initialized")
	}
	db.state.mu.Lock()
	defer db.state.mu.Unlock()
	if db.state.conn != nil {
		return db.state.conn, nil
	}
	conn, err := pgx.Connect(ctx, db.state.url)
	if err != nil {
		return nil, err
	}
	db.state.conn = conn
	return db.state.conn, nil
}

func (db DB) Close() error {
	if db.state == nil {
		return nil
	}
	db.state.mu.Lock()
	defer db.state.mu.Unlock()

	var firstErr error
	if db.state.conn != nil {
		if err := db.state.conn.Close(context.Background()); err != nil {
			firstErr = err
		}
		db.state.conn = nil
	}
	if db.state.stopFn != nil {
		if err := db.state.stopFn(); err != nil && firstErr == nil {
			firstErr = err
		}
		db.state.stopFn = nil
	}
	return firstErr
}

func WithPg(ctx context.Context, cfg Config, fn func(context.Context, *DB) error) error {
	if fn == nil {
		return fmt.Errorf("withpg: callback is nil")
	}
	configs, err := runConfigs(cfg)
	if err != nil {
		return err
	}

	for _, runCfg := range configs {
		runCtx, cancel := context.WithCancel(ctx)
		db, err := Start(runCtx, runCfg)
		if err != nil {
			cancel()
			return err
		}

		runErr := fn(runCtx, db)
		closeErr := db.Close()
		cancel()
		if runErr != nil {
			return errors.Join(runErr, closeErr)
		}
		if closeErr != nil {
			return closeErr
		}
	}

	return nil
}

func Test(t *testing.T, cfg Config, fn func(*testing.T, DB)) {
	t.Helper()
	if fn == nil {
		t.Fatal("withpg: test callback is nil")
	}
	configs, err := runConfigs(cfg)
	if err != nil {
		t.Fatal(err)
	}

	for _, runCfg := range configs {
		runCfg := runCfg
		t.Run("postgres-"+runCfg.PgVersion, func(t *testing.T) {
			t.Helper()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			db, err := Start(ctx, runCfg)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := db.Close(); err != nil {
					t.Errorf("close PostgreSQL: %v", err)
				}
			}()

			fn(t, *db)
		})
	}
}

func Start(ctx context.Context, cfg Config) (*DB, error) {
	if err := validateStartConfig(cfg); err != nil {
		return nil, err
	}
	pgConfig, connStr, err := postgresConfig(cfg)
	if err != nil {
		return nil, err
	}
	postgres := embeddedpostgres.NewDatabase(pgConfig)

	startMu.Lock()
	err = postgres.Start()
	startMu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("withpg: start PostgreSQL: %w", err)
	}

	db := &DB{
		state: &dbState{
			url: connStr,
			stopFn: func() error {
				return postgres.Stop()
			},
		},
	}
	if cfg.SchemaFile != "" {
		if err := loadSchema(ctx, db, cfg.SchemaFile); err != nil {
			_ = db.Close()
			return nil, err
		}
	}
	return db, nil
}

func loadSchema(ctx context.Context, db *DB, schemaFile string) error {
	schemaSQL, err := os.ReadFile(schemaFile)
	if err != nil {
		return fmt.Errorf("withpg: read schema file: %w", err)
	}
	conn, err := db.Client(ctx)
	if err != nil {
		return fmt.Errorf("withpg: connect for schema load: %w", err)
	}
	if _, err := conn.Exec(ctx, string(schemaSQL)); err != nil {
		return fmt.Errorf("withpg: execute schema: %w", err)
	}
	return nil
}

func runConfigs(cfg Config) ([]Config, error) {
	if cfg.PgVersion != "" && len(cfg.PgVersions) > 0 {
		return nil, fmt.Errorf("withpg: set either PgVersion or PgVersions, not both")
	}
	if len(cfg.PgVersions) == 0 {
		version := cfg.PgVersion
		if version == "" {
			version = "18"
		}
		if _, err := parseVersion(version); err != nil {
			return nil, err
		}
		cfg.PgVersion = version
		return []Config{cfg}, nil
	}

	configs := make([]Config, 0, len(cfg.PgVersions))
	for _, version := range cfg.PgVersions {
		if version == "" {
			version = "18"
		}
		if _, err := parseVersion(version); err != nil {
			return nil, err
		}
		runCfg := cfg
		runCfg.PgVersion = version
		runCfg.PgVersions = nil
		configs = append(configs, runCfg)
	}
	return configs, nil
}

func validateStartConfig(cfg Config) error {
	if len(cfg.PgVersions) == 0 {
		return nil
	}
	if cfg.PgVersion != "" {
		return fmt.Errorf("withpg: set either PgVersion or PgVersions, not both")
	}
	return fmt.Errorf("withpg: Start does not accept PgVersions; use WithPg or Test")
}

func postgresConfig(cfg Config) (embeddedpostgres.Config, string, error) {
	if err := validateStartConfig(cfg); err != nil {
		return embeddedpostgres.Config{}, "", err
	}
	port := cfg.Port
	if port == 0 {
		availablePort, err := availablePort()
		if err != nil {
			return embeddedpostgres.Config{}, "", err
		}
		port = uint32(availablePort)
	}

	version, err := parseVersion(cfg.PgVersion)
	if err != nil {
		return embeddedpostgres.Config{}, "", err
	}
	dbName := cfg.DBName
	if dbName == "" {
		dbName = "testdb"
	}
	logger := cfg.Logger
	if logger == nil {
		logger = io.Discard
	}

	pgConfig := embeddedpostgres.DefaultConfig().
		Version(version).
		Port(port).
		Database(dbName).
		Username("postgres").
		Password("postgres").
		Logger(logger)
	if cfg.DataPath != "" {
		pgConfig = pgConfig.DataPath(cfg.DataPath)
	}

	connStr := fmt.Sprintf("postgres://postgres:postgres@localhost:%d/%s?sslmode=disable", port, dbName)
	return pgConfig, connStr, nil
}

func availablePort() (int, error) {
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return 0, fmt.Errorf("withpg: reserve port: %w", err)
	}
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	return addr.Port, nil
}

func parseVersion(version string) (embeddedpostgres.PostgresVersion, error) {
	if version == "" {
		version = "18"
	}
	switch version {
	case "9":
		return embeddedpostgres.V9, nil
	case "10":
		return embeddedpostgres.V10, nil
	case "11":
		return embeddedpostgres.V11, nil
	case "12":
		return embeddedpostgres.V12, nil
	case "13":
		return embeddedpostgres.V13, nil
	case "14":
		return embeddedpostgres.V14, nil
	case "15":
		return embeddedpostgres.V15, nil
	case "16":
		return embeddedpostgres.V16, nil
	case "17":
		return embeddedpostgres.V17, nil
	case "18":
		return embeddedpostgres.V18, nil
	default:
		return embeddedpostgres.V18, fmt.Errorf("withpg: unsupported PostgreSQL version %q", version)
	}
}
