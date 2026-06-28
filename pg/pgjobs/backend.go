package pgjobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golazy.dev/lazyjobs"
)

type Backend struct {
	pool *pgxpool.Pool
}

var _ lazyjobs.Backend = (*Backend)(nil)

func New(pool *pgxpool.Pool) *Backend {
	return &Backend{pool: pool}
}

func (backend *Backend) Insert(ctx context.Context, params lazyjobs.InsertParams) (lazyjobs.Record, error) {
	if err := backend.validate(); err != nil {
		return lazyjobs.Record{}, err
	}
	runAt := params.RunAt
	if runAt.IsZero() {
		runAt = time.Now().UTC()
	}
	return scanRecord(backend.pool.QueryRow(ctx, `
INSERT INTO lazy_jobs (
	kind,
	queue,
	payload,
	state,
	max_attempts,
	run_at
) VALUES (
	$1, $2, $3, 'pending', $4, $5
)
RETURNING id, kind, queue, payload, state, attempt, max_attempts, run_at, created_at, updated_at, last_error`,
		params.Kind,
		normalizeQueue(params.Queue),
		params.Payload,
		normalizeAttempts(params.MaxAttempts),
		runAt,
	))
}

func (backend *Backend) Claim(ctx context.Context, params lazyjobs.ClaimParams) (lazyjobs.Record, bool, error) {
	if err := backend.validate(); err != nil {
		return lazyjobs.Record{}, false, err
	}
	now := params.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	record, err := scanRecord(backend.pool.QueryRow(ctx, `
WITH candidate AS (
	SELECT id
	FROM lazy_jobs
	WHERE queue = ANY($1::text[])
	  AND state IN ('pending', 'retrying')
	  AND run_at <= $2
	ORDER BY run_at ASC, id ASC
	LIMIT 1
	FOR UPDATE SKIP LOCKED
)
UPDATE lazy_jobs
SET state = 'running',
	attempt = attempt + 1,
	locked_at = $2,
	updated_at = $2
WHERE id = (SELECT id FROM candidate)
RETURNING id, kind, queue, payload, state, attempt, max_attempts, run_at, created_at, updated_at, last_error`,
		normalizeQueues(params.Queues),
		now,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return lazyjobs.Record{}, false, nil
	}
	if err != nil {
		return lazyjobs.Record{}, false, err
	}
	return record, true, nil
}

func (backend *Backend) Complete(ctx context.Context, id int64) error {
	if err := backend.validate(); err != nil {
		return err
	}
	_, err := backend.pool.Exec(ctx, `
UPDATE lazy_jobs
SET state = 'succeeded',
	updated_at = now()
WHERE id = $1`, id)
	return err
}

func (backend *Backend) Retry(ctx context.Context, params lazyjobs.RetryParams) error {
	if err := backend.validate(); err != nil {
		return err
	}
	_, err := backend.pool.Exec(ctx, `
UPDATE lazy_jobs
SET state = 'retrying',
	run_at = $2,
	last_error = $3,
	updated_at = now()
WHERE id = $1`, params.ID, params.RunAt, params.LastError)
	return err
}

func (backend *Backend) Discard(ctx context.Context, params lazyjobs.DiscardParams) error {
	if err := backend.validate(); err != nil {
		return err
	}
	_, err := backend.pool.Exec(ctx, `
UPDATE lazy_jobs
SET state = 'discarded',
	last_error = $2,
	updated_at = now()
WHERE id = $1`, params.ID, params.LastError)
	return err
}

func (backend *Backend) List(ctx context.Context, options lazyjobs.ListOptions) ([]lazyjobs.Record, error) {
	if err := backend.validate(); err != nil {
		return nil, err
	}
	limit := options.Limit
	if limit <= 0 {
		limit = 100
	}
	rows, err := backend.pool.Query(ctx, `
SELECT id, kind, queue, payload, state, attempt, max_attempts, run_at, created_at, updated_at, last_error
FROM lazy_jobs
ORDER BY updated_at DESC, id DESC
LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := []lazyjobs.Record{}
	for rows.Next() {
		record, err := scanRecord(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return records, nil
}

func (backend *Backend) Stats(ctx context.Context) (lazyjobs.Stats, error) {
	if err := backend.validate(); err != nil {
		return lazyjobs.Stats{}, err
	}
	stats := lazyjobs.Stats{
		ByState: map[lazyjobs.State]int{},
		ByKind:  map[string]int{},
		ByQueue: map[string]int{},
	}
	if err := countInto(ctx, backend.pool, `SELECT state, COUNT(*)::bigint FROM lazy_jobs GROUP BY state`, func(key string, count int) {
		stats.Total += count
		stats.ByState[lazyjobs.State(key)] = count
	}); err != nil {
		return lazyjobs.Stats{}, err
	}
	if err := countInto(ctx, backend.pool, `SELECT kind, COUNT(*)::bigint FROM lazy_jobs GROUP BY kind`, func(key string, count int) {
		stats.ByKind[key] = count
	}); err != nil {
		return lazyjobs.Stats{}, err
	}
	if err := countInto(ctx, backend.pool, `SELECT queue, COUNT(*)::bigint FROM lazy_jobs GROUP BY queue`, func(key string, count int) {
		stats.ByQueue[key] = count
	}); err != nil {
		return lazyjobs.Stats{}, err
	}
	return stats, nil
}

func (backend *Backend) validate() error {
	if backend == nil || backend.pool == nil {
		return fmt.Errorf("pgjobs: pgx pool is required")
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanRecord(row rowScanner) (lazyjobs.Record, error) {
	var record lazyjobs.Record
	var payload []byte
	var state string
	var attempt int32
	var maxAttempts int32
	if err := row.Scan(
		&record.ID,
		&record.Kind,
		&record.Queue,
		&payload,
		&state,
		&attempt,
		&maxAttempts,
		&record.RunAt,
		&record.CreatedAt,
		&record.UpdatedAt,
		&record.LastError,
	); err != nil {
		return lazyjobs.Record{}, err
	}
	record.Payload = append(json.RawMessage(nil), payload...)
	record.State = lazyjobs.State(state)
	record.Attempt = int(attempt)
	record.MaxAttempts = int(maxAttempts)
	return record, nil
}

func countInto(ctx context.Context, pool *pgxpool.Pool, sql string, fn func(string, int)) error {
	rows, err := pool.Query(ctx, sql)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var key string
		var count int64
		if err := rows.Scan(&key, &count); err != nil {
			return err
		}
		fn(key, int(count))
	}
	return rows.Err()
}

func normalizeQueue(queue string) string {
	if queue == "" {
		return lazyjobs.DefaultQueue
	}
	return queue
}

func normalizeQueues(queues []string) []string {
	if len(queues) == 0 {
		return []string{lazyjobs.DefaultQueue}
	}
	out := make([]string, len(queues))
	for index, queue := range queues {
		out[index] = normalizeQueue(queue)
	}
	return out
}

func normalizeAttempts(attempts int) int {
	if attempts <= 0 {
		return 25
	}
	return attempts
}
