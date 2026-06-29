-- +lazy Up
CREATE TABLE lazy_media_variants (
    source_file_id TEXT NOT NULL,
    variant_key TEXT NOT NULL,
    spec JSONB,
    output_file_id TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'ready',
    error TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (source_file_id, variant_key)
);

CREATE INDEX lazy_media_variants_output_file_id_idx
ON lazy_media_variants (output_file_id)
WHERE output_file_id <> '';

-- +lazy Down
DROP TABLE lazy_media_variants;
