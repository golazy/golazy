-- +lazy Up
CREATE TABLE lazy_files (
    id TEXT PRIMARY KEY,
    filename TEXT NOT NULL DEFAULT '',
    content_type TEXT NOT NULL DEFAULT '',
    size BIGINT NOT NULL DEFAULT 0,
    checksum TEXT NOT NULL DEFAULT '',
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE lazy_file_locations (
    file_id TEXT NOT NULL REFERENCES lazy_files(id) ON DELETE CASCADE,
    storage TEXT NOT NULL,
    key TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'primary',
    status TEXT NOT NULL DEFAULT 'active',
    checksum TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (file_id, storage, key)
);

CREATE INDEX lazy_file_locations_file_id_idx
ON lazy_file_locations (file_id, status, role, created_at);

-- +lazy Down
DROP TABLE lazy_file_locations;
DROP TABLE lazy_files;
