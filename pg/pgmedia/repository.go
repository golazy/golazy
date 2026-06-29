package pgmedia

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golazy.dev/lazymedia"
)

type Repository struct {
	pool *pgxpool.Pool
}

var _ lazymedia.Repository = (*Repository)(nil)

// New creates a PostgreSQL-backed lazymedia repository.
func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (repo *Repository) FindVariant(ctx context.Context, sourceFileID, variantKey string, options ...any) (lazymedia.Variant, []any, error) {
	if err := repo.validate(); err != nil {
		return lazymedia.Variant{}, options, err
	}
	if err := ctxErr(ctx); err != nil {
		return lazymedia.Variant{}, options, err
	}
	variant, err := scanVariant(repo.pool.QueryRow(ctx, `
SELECT source_file_id, variant_key, spec, output_file_id, status, error, created_at, updated_at
FROM lazy_media_variants
WHERE source_file_id = $1
  AND variant_key = $2`,
		sourceFileID,
		variantKey,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return lazymedia.Variant{}, options, errNotExist(sourceFileID, variantKey)
	}
	if err != nil {
		return lazymedia.Variant{}, options, fmt.Errorf("pgmedia: find variant %s/%s: %w", sourceFileID, variantKey, err)
	}
	return variant, options, nil
}

func (repo *Repository) SaveVariant(ctx context.Context, variant lazymedia.Variant, options ...any) (lazymedia.Variant, []any, error) {
	if err := repo.validate(); err != nil {
		return lazymedia.Variant{}, options, err
	}
	if err := ctxErr(ctx); err != nil {
		return lazymedia.Variant{}, options, err
	}
	if err := validateVariant(variant); err != nil {
		return lazymedia.Variant{}, options, err
	}
	if variant.Status == "" {
		variant.Status = lazymedia.StatusReady
	}
	now := time.Now().UTC()
	saved, err := scanVariant(repo.pool.QueryRow(ctx, `
INSERT INTO lazy_media_variants (
	source_file_id,
	variant_key,
	spec,
	output_file_id,
	status,
	error,
	created_at,
	updated_at
) VALUES (
	$1, $2, $3::jsonb, $4, $5, $6, COALESCE($7::timestamptz, $8), $8
)
ON CONFLICT (source_file_id, variant_key) DO UPDATE
SET spec = EXCLUDED.spec,
	output_file_id = EXCLUDED.output_file_id,
	status = EXCLUDED.status,
	error = EXCLUDED.error,
	updated_at = EXCLUDED.updated_at
RETURNING source_file_id, variant_key, spec, output_file_id, status, error, created_at, updated_at`,
		variant.SourceFileID,
		variant.VariantKey,
		nullableJSON(variant.Spec),
		variant.OutputFileID,
		variant.Status,
		variant.Error,
		nullableTime(variant.CreatedAt),
		now,
	))
	if err != nil {
		return lazymedia.Variant{}, options, fmt.Errorf("pgmedia: save variant %s/%s: %w", variant.SourceFileID, variant.VariantKey, err)
	}
	return saved, options, nil
}

func (repo *Repository) DeleteVariant(ctx context.Context, sourceFileID, variantKey string, options ...any) ([]any, error) {
	if err := repo.validate(); err != nil {
		return options, err
	}
	if err := ctxErr(ctx); err != nil {
		return options, err
	}
	if _, err := repo.pool.Exec(ctx, `
DELETE FROM lazy_media_variants
WHERE source_file_id = $1
  AND variant_key = $2`,
		sourceFileID,
		variantKey,
	); err != nil {
		return options, fmt.Errorf("pgmedia: delete variant %s/%s: %w", sourceFileID, variantKey, err)
	}
	return options, nil
}

func (repo *Repository) validate() error {
	if repo == nil || repo.pool == nil {
		return fmt.Errorf("pgmedia: pgx pool is required")
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanVariant(row rowScanner) (lazymedia.Variant, error) {
	var variant lazymedia.Variant
	var spec []byte
	if err := row.Scan(
		&variant.SourceFileID,
		&variant.VariantKey,
		&spec,
		&variant.OutputFileID,
		&variant.Status,
		&variant.Error,
		&variant.CreatedAt,
		&variant.UpdatedAt,
	); err != nil {
		return lazymedia.Variant{}, err
	}
	if len(spec) > 0 {
		variant.Spec = append(json.RawMessage(nil), spec...)
	}
	return variant, nil
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

func validateVariant(variant lazymedia.Variant) error {
	if variant.SourceFileID == "" {
		return fmt.Errorf("lazymedia: source file id is required")
	}
	if variant.VariantKey == "" {
		return fmt.Errorf("lazymedia: variant key is required")
	}
	return nil
}

func errNotExist(sourceFileID, variantKey string) error {
	return fmt.Errorf("lazymedia: variant %q/%q not found: %w", sourceFileID, variantKey, os.ErrNotExist)
}

func ctxErr(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}
