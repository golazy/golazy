package pgstorage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golazy.dev/lazystorage"
)

type Storage struct {
	pool *pgxpool.Pool
}

var _ lazystorage.Storage = (*Storage)(nil)
var _ lazystorage.Writer = (*Storage)(nil)
var _ lazystorage.Deleter = (*Storage)(nil)
var _ lazystorage.Lister = (*Storage)(nil)

// New creates a PostgreSQL-backed object store.
func New(pool *pgxpool.Pool) *Storage {
	return &Storage{pool: pool}
}

func (storage *Storage) Open(ctx context.Context, key string, options ...any) (lazystorage.File, []any, error) {
	if err := storage.validate(); err != nil {
		return nil, options, err
	}
	if err := ctxErr(ctx); err != nil {
		return nil, options, err
	}
	if err := lazystorage.ValidateKey(key); err != nil {
		return nil, options, err
	}
	var data []byte
	var info lazystorage.Info
	err := storage.pool.QueryRow(ctx, `
SELECT key, content, content_type, size, checksum, updated_at
FROM lazy_storage_objects
WHERE key = $1`,
		key,
	).Scan(
		&info.Key,
		&data,
		&info.ContentType,
		&info.Size,
		&info.Checksum,
		&info.ModifiedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, options, errNotExist(key)
	}
	if err != nil {
		return nil, options, fmt.Errorf("pgstorage: open %s: %w", key, err)
	}
	return &objectFile{
		Reader: bytes.NewReader(data),
		info:   info,
	}, options, nil
}

func (storage *Storage) Put(ctx context.Context, key string, body io.Reader, options ...any) (lazystorage.Info, []any, error) {
	if err := storage.validate(); err != nil {
		return lazystorage.Info{}, options, err
	}
	if err := ctxErr(ctx); err != nil {
		return lazystorage.Info{}, options, err
	}
	if err := lazystorage.ValidateKey(key); err != nil {
		return lazystorage.Info{}, options, err
	}
	if body == nil {
		return lazystorage.Info{}, options, fmt.Errorf("pgstorage: nil body")
	}
	contentType, remaining, _ := lazystorage.Take[lazystorage.ContentType](options)
	cacheControl, remaining, _ := lazystorage.Take[lazystorage.CacheControl](remaining)
	disposition, remaining, _ := lazystorage.Take[lazystorage.ContentDisposition](remaining)

	data, err := io.ReadAll(body)
	if err != nil {
		return lazystorage.Info{}, remaining, fmt.Errorf("pgstorage: read %s: %w", key, err)
	}
	detected := strings.TrimSpace(contentType.Value)
	if detected == "" {
		detected = contentTypeForKey(key, data)
	}
	sum := sha256.Sum256(data)
	checksum := "sha256:" + hex.EncodeToString(sum[:])
	now := time.Now().UTC()

	var info lazystorage.Info
	if err := storage.pool.QueryRow(ctx, `
INSERT INTO lazy_storage_objects (
	key,
	content,
	content_type,
	size,
	checksum,
	cache_control,
	content_disposition,
	created_at,
	updated_at
) VALUES (
	$1, $2, $3, $4, $5, $6, $7, $8, $8
)
ON CONFLICT (key) DO UPDATE
SET content = EXCLUDED.content,
	content_type = EXCLUDED.content_type,
	size = EXCLUDED.size,
	checksum = EXCLUDED.checksum,
	cache_control = EXCLUDED.cache_control,
	content_disposition = EXCLUDED.content_disposition,
	updated_at = EXCLUDED.updated_at
RETURNING key, content_type, size, checksum, updated_at`,
		key,
		data,
		detected,
		int64(len(data)),
		checksum,
		cacheControl.Value,
		disposition.Value,
		now,
	).Scan(
		&info.Key,
		&info.ContentType,
		&info.Size,
		&info.Checksum,
		&info.ModifiedAt,
	); err != nil {
		return lazystorage.Info{}, remaining, fmt.Errorf("pgstorage: put %s: %w", key, err)
	}
	return info, remaining, nil
}

func (storage *Storage) Delete(ctx context.Context, key string, options ...any) ([]any, error) {
	if err := storage.validate(); err != nil {
		return options, err
	}
	if err := ctxErr(ctx); err != nil {
		return options, err
	}
	if err := lazystorage.ValidateKey(key); err != nil {
		return options, err
	}
	if _, err := storage.pool.Exec(ctx, `DELETE FROM lazy_storage_objects WHERE key = $1`, key); err != nil {
		return options, fmt.Errorf("pgstorage: delete %s: %w", key, err)
	}
	return options, nil
}

func (storage *Storage) List(ctx context.Context, prefix string, options ...any) (lazystorage.Iterator, []any, error) {
	if err := storage.validate(); err != nil {
		return nil, options, err
	}
	if err := ctxErr(ctx); err != nil {
		return nil, options, err
	}
	if prefix != "" {
		if err := lazystorage.ValidateKey(prefix); err != nil {
			return nil, options, err
		}
	}
	rows, err := storage.pool.Query(ctx, `
SELECT key, content_type, size, checksum, updated_at
FROM lazy_storage_objects
WHERE $1 = ''
   OR key LIKE $1 || '%'
ORDER BY key ASC`,
		prefix,
	)
	if err != nil {
		return nil, options, fmt.Errorf("pgstorage: list %s: %w", prefix, err)
	}
	defer rows.Close()

	infos := []lazystorage.Info{}
	for rows.Next() {
		var info lazystorage.Info
		if err := rows.Scan(
			&info.Key,
			&info.ContentType,
			&info.Size,
			&info.Checksum,
			&info.ModifiedAt,
		); err != nil {
			return nil, options, fmt.Errorf("pgstorage: scan list %s: %w", prefix, err)
		}
		infos = append(infos, info)
	}
	if err := rows.Err(); err != nil {
		return nil, options, fmt.Errorf("pgstorage: read list %s: %w", prefix, err)
	}
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Key < infos[j].Key
	})
	return &sliceIterator{infos: infos}, options, nil
}

func (storage *Storage) validate() error {
	if storage == nil || storage.pool == nil {
		return fmt.Errorf("pgstorage: pgx pool is required")
	}
	return nil
}

type objectFile struct {
	*bytes.Reader
	info lazystorage.Info
}

func (file *objectFile) Close() error {
	return nil
}

func (file *objectFile) Stat() (lazystorage.Info, error) {
	return file.info, nil
}

type sliceIterator struct {
	infos []lazystorage.Info
	index int
}

func (iterator *sliceIterator) Next() (lazystorage.Info, error) {
	if iterator.index >= len(iterator.infos) {
		return lazystorage.Info{}, io.EOF
	}
	info := iterator.infos[iterator.index]
	iterator.index++
	return info, nil
}

func (iterator *sliceIterator) Close() error {
	return nil
}

func errNotExist(key string) error {
	return fmt.Errorf("lazystorage: object %q not found: %w", key, os.ErrNotExist)
}

func contentTypeForKey(key string, data []byte) string {
	if contentType := mime.TypeByExtension(path.Ext(key)); contentType != "" {
		return contentType
	}
	if len(data) != 0 {
		return http.DetectContentType(data)
	}
	return "application/octet-stream"
}

func ctxErr(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}
