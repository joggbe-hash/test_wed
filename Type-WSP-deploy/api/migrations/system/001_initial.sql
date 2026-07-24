CREATE TABLE IF NOT EXISTS posts (
    id                    SERIAL       PRIMARY KEY,
    user_id               INTEGER      NOT NULL,
    username              VARCHAR(50)  NOT NULL,
    content               TEXT,
    image_url             TEXT,
    image_status          VARCHAR(20)  NOT NULL DEFAULT 'none',
    processing_token      UUID,
    processing_started_at TIMESTAMPTZ,
    created_at            TIMESTAMP    NOT NULL DEFAULT NOW()
);

ALTER TABLE posts
    ADD COLUMN IF NOT EXISTS processing_token UUID;

ALTER TABLE posts
    ADD COLUMN IF NOT EXISTS processing_started_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_posts_feed_cursor
    ON posts (created_at DESC, id DESC);
