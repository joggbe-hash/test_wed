CREATE TABLE IF NOT EXISTS image_deletion_jobs (
    id                    UUID        PRIMARY KEY,
    user_id               INTEGER     NOT NULL,
    object_keys           JSONB       NOT NULL,
    reserved_bytes        BIGINT      NOT NULL,
    processing_token      UUID,
    processing_started_at TIMESTAMPTZ,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT image_deletion_jobs_object_keys_array
        CHECK (jsonb_typeof(object_keys) = 'array'),
    CONSTRAINT image_deletion_jobs_reserved_bytes_nonnegative
        CHECK (reserved_bytes >= 0)
);

CREATE INDEX IF NOT EXISTS idx_image_deletion_jobs_owner
    ON image_deletion_jobs (user_id)
    INCLUDE (reserved_bytes);

CREATE INDEX IF NOT EXISTS idx_image_deletion_jobs_claim
    ON image_deletion_jobs (processing_started_at, created_at);
