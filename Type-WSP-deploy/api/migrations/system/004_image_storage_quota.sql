ALTER TABLE posts
    ADD COLUMN IF NOT EXISTS image_reserved_bytes BIGINT NOT NULL DEFAULT 0;

ALTER TABLE posts
    ADD COLUMN IF NOT EXISTS image_storage_bytes BIGINT NOT NULL DEFAULT 0;

UPDATE posts
SET image_storage_bytes = LEAST(jsonb_array_length(image_url::jsonb), 4) * 16777216
WHERE image_status = 'ready'
  AND image_url IS NOT NULL
  AND image_url <> ''
  AND image_storage_bytes = 0;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'posts_image_reserved_bytes_nonnegative'
    ) THEN
        ALTER TABLE posts
            ADD CONSTRAINT posts_image_reserved_bytes_nonnegative
            CHECK (image_reserved_bytes >= 0);
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'posts_image_storage_bytes_nonnegative'
    ) THEN
        ALTER TABLE posts
            ADD CONSTRAINT posts_image_storage_bytes_nonnegative
            CHECK (image_storage_bytes >= 0);
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_posts_owner_image_storage
    ON posts (user_id)
    INCLUDE (image_status, image_reserved_bytes, image_storage_bytes);
