-- +lazy Up
CREATE TABLE lazy_jobs (
    id BIGSERIAL PRIMARY KEY,
    kind TEXT NOT NULL,
    queue TEXT NOT NULL DEFAULT 'default',
    payload JSONB NOT NULL,
    state TEXT NOT NULL,
    attempt INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL,
    run_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    locked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_error TEXT NOT NULL DEFAULT ''
);

CREATE INDEX lazy_jobs_claim_idx
ON lazy_jobs (queue, state, run_at, id)
WHERE state IN ('pending', 'retrying');

CREATE INDEX lazy_jobs_updated_at_idx
ON lazy_jobs (updated_at DESC, id DESC);

-- +lazy Down
DROP TABLE lazy_jobs;
