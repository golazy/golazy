-- +lazy Up
CREATE TABLE lazy_storage_objects (
    key TEXT PRIMARY KEY,
    content BYTEA NOT NULL,
    content_type TEXT NOT NULL DEFAULT '',
    size BIGINT NOT NULL DEFAULT 0,
    checksum TEXT NOT NULL DEFAULT '',
    cache_control TEXT NOT NULL DEFAULT '',
    content_disposition TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX lazy_storage_objects_prefix_idx
ON lazy_storage_objects (key text_pattern_ops);

-- +lazy Down
DROP TABLE lazy_storage_objects;
