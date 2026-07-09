-- +lazy Up
ALTER TABLE lazy_jobs
ADD COLUMN IF NOT EXISTS schedule_key TEXT;

CREATE INDEX IF NOT EXISTS lazy_jobs_schedule_active_idx
ON lazy_jobs (schedule_key, state)
WHERE schedule_key IS NOT NULL
  AND state IN ('pending', 'retrying', 'running');

CREATE INDEX IF NOT EXISTS lazy_jobs_running_queue_idx
ON lazy_jobs (queue, state)
WHERE state = 'running';

CREATE TABLE IF NOT EXISTS lazy_job_schedules (
    key TEXT PRIMARY KEY,
    kind TEXT NOT NULL,
    queue TEXT NOT NULL DEFAULT 'default',
    payload JSONB NOT NULL,
    interval_ns BIGINT NOT NULL,
    next_run_at TIMESTAMPTZ NOT NULL,
    locked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS lazy_job_schedules_due_idx
ON lazy_job_schedules (next_run_at, key);

-- +lazy Down
DROP INDEX IF EXISTS lazy_job_schedules_due_idx;
DROP TABLE IF EXISTS lazy_job_schedules;
DROP INDEX IF EXISTS lazy_jobs_running_queue_idx;
DROP INDEX IF EXISTS lazy_jobs_schedule_active_idx;
ALTER TABLE lazy_jobs
DROP COLUMN IF EXISTS schedule_key;
