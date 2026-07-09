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
	schedule_key,
	payload,
	state,
	max_attempts,
	run_at
) VALUES (
	$1, $2, $3, $4, 'pending', $5, $6
)
RETURNING id, kind, queue, COALESCE(schedule_key, ''), payload, state, attempt, max_attempts, run_at, created_at, updated_at, last_error`,
		params.Kind,
		normalizeQueue(params.Queue),
		emptyToNil(params.ScheduleKey),
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
	limitQueues, limitValues := queueLimitArrays(params.QueueLimits)
	record, err := scanRecord(backend.pool.QueryRow(ctx, `
WITH limits(queue, max_running) AS (
	SELECT * FROM unnest($3::text[], $4::int[])
),
candidate AS (
	SELECT j.id
	FROM lazy_jobs AS j
	LEFT JOIN limits AS l ON l.queue = j.queue
	WHERE j.queue = ANY($1::text[])
	  AND j.state IN ('pending', 'retrying')
	  AND j.run_at <= $2
	  AND (
		l.max_running IS NULL
		OR (
			SELECT COUNT(*)::int
			FROM lazy_jobs AS running
			WHERE running.queue = j.queue
			  AND running.state = 'running'
		) < l.max_running
	  )
	ORDER BY j.run_at ASC, j.id ASC
	LIMIT 1
	FOR UPDATE SKIP LOCKED
)
UPDATE lazy_jobs
SET state = 'running',
	attempt = attempt + 1,
	locked_at = $2,
	updated_at = $2
WHERE id = (SELECT id FROM candidate)
RETURNING id, kind, queue, COALESCE(schedule_key, ''), payload, state, attempt, max_attempts, run_at, created_at, updated_at, last_error`,
		normalizeQueues(params.Queues),
		now,
		limitQueues,
		limitValues,
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
SELECT id, kind, queue, COALESCE(schedule_key, ''), payload, state, attempt, max_attempts, run_at, created_at, updated_at, last_error
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
		ByState:      map[lazyjobs.State]int{},
		ByKind:       map[string]int{},
		ByQueue:      map[string]int{},
		ByQueueState: map[string]map[lazyjobs.State]int{},
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
	rows, err := backend.pool.Query(ctx, `SELECT queue, state, COUNT(*)::bigint FROM lazy_jobs GROUP BY queue, state`)
	if err != nil {
		return lazyjobs.Stats{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var queue string
		var state string
		var count int64
		if err := rows.Scan(&queue, &state, &count); err != nil {
			return lazyjobs.Stats{}, err
		}
		if stats.ByQueueState[queue] == nil {
			stats.ByQueueState[queue] = map[lazyjobs.State]int{}
		}
		stats.ByQueueState[queue][lazyjobs.State(state)] = int(count)
	}
	if err := rows.Err(); err != nil {
		return lazyjobs.Stats{}, err
	}
	return stats, nil
}

func (backend *Backend) RegisterSchedule(ctx context.Context, params lazyjobs.ScheduleParams) (lazyjobs.ScheduleRecord, error) {
	if err := backend.validate(); err != nil {
		return lazyjobs.ScheduleRecord{}, err
	}
	nextRunAt := params.NextRunAt
	if nextRunAt.IsZero() {
		nextRunAt = time.Now().UTC()
	}
	return scanSchedule(backend.pool.QueryRow(ctx, `
INSERT INTO lazy_job_schedules (
	key,
	kind,
	queue,
	payload,
	interval_ns,
	next_run_at
) VALUES (
	$1, $2, $3, $4, $5, $6
)
ON CONFLICT (key) DO UPDATE SET
	kind = EXCLUDED.kind,
	queue = EXCLUDED.queue,
	payload = EXCLUDED.payload,
	interval_ns = EXCLUDED.interval_ns,
	locked_at = NULL,
	updated_at = now()
RETURNING key, kind, queue, payload, interval_ns, next_run_at, created_at, updated_at`,
		params.Key,
		params.Kind,
		normalizeQueue(params.Queue),
		params.Payload,
		int64(params.Interval),
		nextRunAt,
	))
}

func (backend *Backend) ClaimSchedule(ctx context.Context, params lazyjobs.ClaimScheduleParams) (lazyjobs.ScheduleRecord, bool, error) {
	if err := backend.validate(); err != nil {
		return lazyjobs.ScheduleRecord{}, false, err
	}
	now := params.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	staleLock := now.Add(-5 * time.Minute)
	record, err := scanSchedule(backend.pool.QueryRow(ctx, `
WITH candidate AS (
	SELECT key
	FROM lazy_job_schedules
	WHERE next_run_at <= $1
	  AND (locked_at IS NULL OR locked_at < $2)
	ORDER BY next_run_at ASC, key ASC
	LIMIT 1
	FOR UPDATE SKIP LOCKED
)
UPDATE lazy_job_schedules
SET locked_at = $1,
	updated_at = $1
WHERE key = (SELECT key FROM candidate)
RETURNING key, kind, queue, payload, interval_ns, next_run_at, created_at, updated_at`,
		now,
		staleLock,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return lazyjobs.ScheduleRecord{}, false, nil
	}
	if err != nil {
		return lazyjobs.ScheduleRecord{}, false, err
	}
	return record, true, nil
}

func (backend *Backend) AdvanceSchedule(ctx context.Context, params lazyjobs.AdvanceScheduleParams) error {
	if err := backend.validate(); err != nil {
		return err
	}
	_, err := backend.pool.Exec(ctx, `
UPDATE lazy_job_schedules
SET next_run_at = $2,
	locked_at = NULL,
	updated_at = now()
WHERE key = $1`, params.Key, params.NextRunAt)
	return err
}

func (backend *Backend) HasActiveScheduledJob(ctx context.Context, params lazyjobs.ActiveScheduledJobParams) (bool, error) {
	if err := backend.validate(); err != nil {
		return false, err
	}
	var exists bool
	err := backend.pool.QueryRow(ctx, `
SELECT EXISTS (
	SELECT 1
	FROM lazy_jobs
	WHERE schedule_key = $1
	  AND state IN ('pending', 'retrying', 'running')
)`, params.ScheduleKey).Scan(&exists)
	return exists, err
}

func (backend *Backend) ListSchedules(ctx context.Context) ([]lazyjobs.ScheduleRecord, error) {
	if err := backend.validate(); err != nil {
		return nil, err
	}
	rows, err := backend.pool.Query(ctx, `
SELECT key, kind, queue, payload, interval_ns, next_run_at, created_at, updated_at
FROM lazy_job_schedules
ORDER BY key ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := []lazyjobs.ScheduleRecord{}
	for rows.Next() {
		record, err := scanSchedule(rows)
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
	var scheduleKey string
	var attempt int32
	var maxAttempts int32
	if err := row.Scan(
		&record.ID,
		&record.Kind,
		&record.Queue,
		&scheduleKey,
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
	record.ScheduleKey = scheduleKey
	record.Payload = append(json.RawMessage(nil), payload...)
	record.State = lazyjobs.State(state)
	record.Attempt = int(attempt)
	record.MaxAttempts = int(maxAttempts)
	return record, nil
}

func scanSchedule(row rowScanner) (lazyjobs.ScheduleRecord, error) {
	var record lazyjobs.ScheduleRecord
	var payload []byte
	var intervalNS int64
	if err := row.Scan(
		&record.Key,
		&record.Kind,
		&record.Queue,
		&payload,
		&intervalNS,
		&record.NextRunAt,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		return lazyjobs.ScheduleRecord{}, err
	}
	record.Payload = append(json.RawMessage(nil), payload...)
	record.Interval = time.Duration(intervalNS)
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

func queueLimitArrays(limits map[string]int) ([]string, []int) {
	if len(limits) == 0 {
		return []string{}, []int{}
	}
	queues := make([]string, 0, len(limits))
	values := make([]int, 0, len(limits))
	for queue, limit := range limits {
		if limit <= 0 {
			continue
		}
		queues = append(queues, normalizeQueue(queue))
		values = append(values, limit)
	}
	return queues, values
}

func normalizeAttempts(attempts int) int {
	if attempts <= 0 {
		return 25
	}
	return attempts
}

func emptyToNil(value string) any {
	if value == "" {
		return nil
	}
	return value
}
