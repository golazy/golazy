package pgfiles

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golazy.dev/lazyfiles"
)

type Repository struct {
	pool *pgxpool.Pool
}

var _ lazyfiles.Repository = (*Repository)(nil)
var _ lazyfiles.Lister = (*Repository)(nil)

// New creates a PostgreSQL-backed lazyfiles repository.
func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (repo *Repository) Put(ctx context.Context, file lazyfiles.File, location lazyfiles.Location, options ...any) (lazyfiles.File, []any, error) {
	if err := repo.validate(); err != nil {
		return lazyfiles.File{}, options, err
	}
	if err := ctxErr(ctx); err != nil {
		return lazyfiles.File{}, options, err
	}
	if location.FileID == "" {
		location.FileID = file.ID
	}
	if location.Role == "" {
		location.Role = lazyfiles.RolePrimary
	}
	if location.Status == "" {
		location.Status = lazyfiles.StatusActive
	}
	if err := validateFileLocation(file, location); err != nil {
		return lazyfiles.File{}, options, err
	}

	now := time.Now().UTC()
	tx, err := repo.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return lazyfiles.File{}, options, fmt.Errorf("pgfiles: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	saved, err := scanFile(tx.QueryRow(ctx, `
INSERT INTO lazy_files (
	id,
	filename,
	content_type,
	size,
	checksum,
	metadata,
	created_at,
	updated_at,
	deleted_at
) VALUES (
	$1, $2, $3, $4, $5, $6::jsonb, COALESCE($7::timestamptz, $8), $8, $9::timestamptz
)
ON CONFLICT (id) DO UPDATE
SET filename = EXCLUDED.filename,
	content_type = EXCLUDED.content_type,
	size = EXCLUDED.size,
	checksum = EXCLUDED.checksum,
	metadata = EXCLUDED.metadata,
	updated_at = EXCLUDED.updated_at,
	deleted_at = EXCLUDED.deleted_at
RETURNING id, filename, content_type, size, checksum, metadata, created_at, updated_at, deleted_at`,
		file.ID,
		file.Filename,
		file.ContentType,
		file.Size,
		file.Checksum,
		nullableJSON(file.Metadata),
		nullableTime(file.CreatedAt),
		now,
		nullableTime(file.DeletedAt),
	))
	if err != nil {
		return lazyfiles.File{}, options, fmt.Errorf("pgfiles: save file %s: %w", file.ID, err)
	}

	if _, err := tx.Exec(ctx, `
INSERT INTO lazy_file_locations (
	file_id,
	storage,
	key,
	role,
	status,
	checksum,
	created_at,
	updated_at
) VALUES (
	$1, $2, $3, $4, $5, $6, $7, $7
)
ON CONFLICT (file_id, storage, key) DO UPDATE
SET role = EXCLUDED.role,
	status = EXCLUDED.status,
	checksum = EXCLUDED.checksum,
	updated_at = EXCLUDED.updated_at`,
		location.FileID,
		location.Storage,
		location.Key,
		location.Role,
		location.Status,
		location.Checksum,
		now,
	); err != nil {
		return lazyfiles.File{}, options, fmt.Errorf("pgfiles: save location %s/%s: %w", location.Storage, location.Key, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return lazyfiles.File{}, options, fmt.Errorf("pgfiles: commit file %s: %w", file.ID, err)
	}
	return saved, options, nil
}

func (repo *Repository) Find(ctx context.Context, query lazyfiles.Query, options ...any) (lazyfiles.File, []lazyfiles.Location, []any, error) {
	if err := repo.validate(); err != nil {
		return lazyfiles.File{}, nil, options, err
	}
	if err := ctxErr(ctx); err != nil {
		return lazyfiles.File{}, nil, options, err
	}
	if query.ID == "" {
		return lazyfiles.File{}, nil, options, fmt.Errorf("lazyfiles: file id is required")
	}

	file, err := scanFile(repo.pool.QueryRow(ctx, `
SELECT id, filename, content_type, size, checksum, metadata, created_at, updated_at, deleted_at
FROM lazy_files
WHERE id = $1
  AND deleted_at IS NULL`,
		query.ID,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return lazyfiles.File{}, nil, options, errNotExist(query.ID)
	}
	if err != nil {
		return lazyfiles.File{}, nil, options, fmt.Errorf("pgfiles: find file %s: %w", query.ID, err)
	}

	locations, err := repo.listLocations(ctx, query.ID)
	if err != nil {
		return lazyfiles.File{}, nil, options, err
	}
	return file, locations, options, nil
}

func (repo *Repository) List(ctx context.Context, query lazyfiles.ListQuery, options ...any) ([]lazyfiles.StoredFile, []any, error) {
	if err := repo.validate(); err != nil {
		return nil, options, err
	}
	if err := ctxErr(ctx); err != nil {
		return nil, options, err
	}
	rows, err := repo.pool.Query(ctx, `
SELECT id, filename, content_type, size, checksum, metadata, created_at, updated_at, deleted_at
FROM lazy_files
WHERE ($1::boolean OR deleted_at IS NULL)
ORDER BY updated_at DESC, id ASC`,
		query.IncludeDeleted,
	)
	if err != nil {
		return nil, options, fmt.Errorf("pgfiles: list files: %w", err)
	}
	defer rows.Close()

	files := []lazyfiles.StoredFile{}
	for rows.Next() {
		file, err := scanFile(rows)
		if err != nil {
			return nil, options, fmt.Errorf("pgfiles: scan file: %w", err)
		}
		locations, err := repo.listLocations(ctx, file.ID)
		if err != nil {
			return nil, options, err
		}
		if !locationsMatch(query, locations) {
			continue
		}
		files = append(files, lazyfiles.StoredFile{
			File:      file,
			Locations: locations,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, options, fmt.Errorf("pgfiles: read files: %w", err)
	}
	return files, options, nil
}

func (repo *Repository) listLocations(ctx context.Context, fileID string) ([]lazyfiles.Location, error) {
	rows, err := repo.pool.Query(ctx, `
SELECT file_id, storage, key, role, status, checksum
FROM lazy_file_locations
WHERE file_id = $1
ORDER BY created_at ASC, storage ASC, key ASC`,
		fileID,
	)
	if err != nil {
		return nil, fmt.Errorf("pgfiles: list locations %s: %w", fileID, err)
	}
	defer rows.Close()

	locations := []lazyfiles.Location{}
	for rows.Next() {
		var location lazyfiles.Location
		if err := rows.Scan(
			&location.FileID,
			&location.Storage,
			&location.Key,
			&location.Role,
			&location.Status,
			&location.Checksum,
		); err != nil {
			return nil, fmt.Errorf("pgfiles: scan location %s: %w", fileID, err)
		}
		locations = append(locations, location)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("pgfiles: read locations %s: %w", fileID, err)
	}
	return locations, nil
}

func (repo *Repository) Delete(ctx context.Context, fileID string, options ...any) ([]any, error) {
	if err := repo.validate(); err != nil {
		return options, err
	}
	if err := ctxErr(ctx); err != nil {
		return options, err
	}
	if fileID == "" {
		return options, fmt.Errorf("lazyfiles: file id is required")
	}
	now := time.Now().UTC()
	if _, err := repo.pool.Exec(ctx, `
INSERT INTO lazy_files (
	id,
	created_at,
	updated_at,
	deleted_at
) VALUES (
	$1, $2, $2, $2
)
ON CONFLICT (id) DO UPDATE
SET updated_at = EXCLUDED.updated_at,
	deleted_at = EXCLUDED.deleted_at`,
		fileID,
		now,
	); err != nil {
		return options, fmt.Errorf("pgfiles: delete file %s: %w", fileID, err)
	}
	return options, nil
}

func (repo *Repository) validate() error {
	if repo == nil || repo.pool == nil {
		return fmt.Errorf("pgfiles: pgx pool is required")
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanFile(row rowScanner) (lazyfiles.File, error) {
	var file lazyfiles.File
	var metadata []byte
	var deletedAt pgtype.Timestamptz
	if err := row.Scan(
		&file.ID,
		&file.Filename,
		&file.ContentType,
		&file.Size,
		&file.Checksum,
		&metadata,
		&file.CreatedAt,
		&file.UpdatedAt,
		&deletedAt,
	); err != nil {
		return lazyfiles.File{}, err
	}
	if len(metadata) > 0 {
		file.Metadata = append(json.RawMessage(nil), metadata...)
	}
	if deletedAt.Valid {
		file.DeletedAt = deletedAt.Time
	}
	return file, nil
}

func nullableJSON(value json.RawMessage) any {
	if len(value) == 0 {
		return nil
	}
	return []byte(value)
}

func nullableTime(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value
}

func validateFileLocation(file lazyfiles.File, location lazyfiles.Location) error {
	if file.ID == "" {
		return fmt.Errorf("lazyfiles: file id is required")
	}
	if location.FileID == "" {
		location.FileID = file.ID
	}
	if location.FileID != file.ID {
		return fmt.Errorf("lazyfiles: location file id %q does not match file id %q", location.FileID, file.ID)
	}
	if location.Storage == "" {
		return fmt.Errorf("lazyfiles: storage name is required")
	}
	if location.Key == "" {
		return fmt.Errorf("lazyfiles: storage key is required")
	}
	return nil
}

func locationsMatch(query lazyfiles.ListQuery, locations []lazyfiles.Location) bool {
	storage := strings.TrimSpace(query.Storage)
	prefix := strings.TrimSpace(query.KeyPrefix)
	if storage == "" && prefix == "" {
		return true
	}
	for _, location := range locations {
		if storage != "" && location.Storage != storage {
			continue
		}
		if prefix != "" && !strings.HasPrefix(location.Key, prefix) {
			continue
		}
		return true
	}
	return false
}

func errNotExist(id string) error {
	return fmt.Errorf("lazyfiles: file %q not found: %w", id, os.ErrNotExist)
}

func ctxErr(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}
