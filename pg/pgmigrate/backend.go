package pgmigrate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golazy.dev/lazymigrate"
)

// ErrSchemaUnsupported is returned by schema load/dump methods until this
// backend grows pg_dump/pg_restore semantics.
var ErrSchemaUnsupported = errors.New("pgmigrate: schema load and dump are not implemented")

type Backend struct {
	pool *pgxpool.Pool
}

var _ lazymigrate.Backend = (*Backend)(nil)

func New(pool *pgxpool.Pool) *Backend {
	return &Backend{pool: pool}
}

func (backend *Backend) Setup(ctx context.Context) error {
	if err := backend.validate(); err != nil {
		return err
	}
	_, err := backend.pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS lazy_migrations (
	id TEXT PRIMARY KEY,
	checksum TEXT NOT NULL,
	applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
)`)
	if err != nil {
		return fmt.Errorf("pgmigrate: setup metadata table: %w", err)
	}
	return nil
}

func (backend *Backend) List(ctx context.Context) ([]lazymigrate.BackendMigration, error) {
	if err := backend.validate(); err != nil {
		return nil, err
	}
	rows, err := backend.pool.Query(ctx, `
SELECT id
FROM lazy_migrations
ORDER BY id ASC`)
	if err != nil {
		return nil, fmt.Errorf("pgmigrate: list migrations: %w", err)
	}
	defer rows.Close()

	migrations := []lazymigrate.BackendMigration{}
	for rows.Next() {
		var migration lazymigrate.BackendMigration
		if err := rows.Scan(&migration.ID); err != nil {
			return nil, fmt.Errorf("pgmigrate: scan migration: %w", err)
		}
		migrations = append(migrations, migration)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("pgmigrate: read migrations: %w", err)
	}
	return migrations, nil
}

func (backend *Backend) Run(ctx context.Context, step lazymigrate.Step) error {
	if err := backend.validate(); err != nil {
		return err
	}
	if step.Migration.ID == "" {
		return fmt.Errorf("pgmigrate: migration id is required")
	}
	parsed, err := parse(step.Migration.Content)
	if err != nil {
		return fmt.Errorf("pgmigrate: parse %s: %w", step.Migration.ID, err)
	}
	sql, err := parsed.forDirection(step.Direction)
	if err != nil {
		return fmt.Errorf("pgmigrate: migration %s: %w", step.Migration.ID, err)
	}

	tx, err := backend.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("pgmigrate: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, int64(7277710616071001)); err != nil {
		return fmt.Errorf("pgmigrate: acquire migration lock: %w", err)
	}

	switch step.Direction {
	case lazymigrate.DirectionUp:
		if err := ensureNotApplied(ctx, tx, step.Migration.ID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, sql); err != nil {
			return fmt.Errorf("pgmigrate: run up %s: %w", step.Migration.ID, err)
		}
		if _, err := tx.Exec(ctx, `
INSERT INTO lazy_migrations (id, checksum)
VALUES ($1, $2)`, step.Migration.ID, checksum(step.Migration.Content)); err != nil {
			return fmt.Errorf("pgmigrate: record %s: %w", step.Migration.ID, err)
		}
	case lazymigrate.DirectionDown:
		if err := ensureApplied(ctx, tx, step.Migration.ID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, sql); err != nil {
			return fmt.Errorf("pgmigrate: run down %s: %w", step.Migration.ID, err)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM lazy_migrations WHERE id = $1`, step.Migration.ID); err != nil {
			return fmt.Errorf("pgmigrate: forget %s: %w", step.Migration.ID, err)
		}
	default:
		return fmt.Errorf("pgmigrate: unsupported direction %q", step.Direction)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("pgmigrate: commit %s: %w", step.Migration.ID, err)
	}
	return nil
}

func (backend *Backend) DumpSchema(context.Context) ([]byte, error) {
	return nil, ErrSchemaUnsupported
}

func (backend *Backend) LoadSchema(context.Context, []byte) error {
	return ErrSchemaUnsupported
}

func (backend *Backend) validate() error {
	if backend == nil || backend.pool == nil {
		return fmt.Errorf("pgmigrate: pgx pool is required")
	}
	return nil
}

func ensureNotApplied(ctx context.Context, tx pgx.Tx, id string) error {
	var storedID string
	err := tx.QueryRow(ctx, `SELECT id FROM lazy_migrations WHERE id = $1`, id).Scan(&storedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("pgmigrate: check %s: %w", id, err)
	}
	return fmt.Errorf("pgmigrate: migration %s has already been applied", id)
}

func ensureApplied(ctx context.Context, tx pgx.Tx, id string) error {
	var storedID string
	err := tx.QueryRow(ctx, `SELECT id FROM lazy_migrations WHERE id = $1`, id).Scan(&storedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("pgmigrate: migration %s has not been applied", id)
	}
	if err != nil {
		return fmt.Errorf("pgmigrate: check %s: %w", id, err)
	}
	return nil
}

func checksum(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}
