CREATE TABLE IF NOT EXISTS image_upload_reservations (
    id             UUID        PRIMARY KEY,
    user_id        INTEGER     NOT NULL,
    raw_keys       JSONB       NOT NULL,
    reserved_bytes BIGINT      NOT NULL,
    expires_at     TIMESTAMPTZ NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT image_upload_reservations_raw_keys_array
        CHECK (jsonb_typeof(raw_keys) = 'array'),
    CONSTRAINT image_upload_reservations_reserved_bytes_positive
        CHECK (reserved_bytes > 0)
);

CREATE INDEX IF NOT EXISTS idx_image_upload_reservations_owner_expiry
    ON image_upload_reservations (user_id, expires_at)
    INCLUDE (reserved_bytes);

CREATE INDEX IF NOT EXISTS idx_image_upload_reservations_expiry
    ON image_upload_reservations (expires_at);
