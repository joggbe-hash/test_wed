ALTER TABLE posts
    ADD COLUMN IF NOT EXISTS visibility VARCHAR(20) NOT NULL DEFAULT 'public';

ALTER TABLE posts
    ADD CONSTRAINT posts_visibility_check
    CHECK (visibility IN ('public', 'private'));

CREATE INDEX IF NOT EXISTS idx_posts_visibility_feed_cursor
    ON posts (visibility, created_at DESC, id DESC);
