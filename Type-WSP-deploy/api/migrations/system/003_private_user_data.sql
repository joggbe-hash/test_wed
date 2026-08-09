CREATE TABLE IF NOT EXISTS user_schedules (
    user_id    INTEGER     PRIMARY KEY,
    schedule   JSONB       NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS inspirations (
    id          BIGSERIAL   PRIMARY KEY,
    user_id     INTEGER     NOT NULL,
    item_date   DATE        NOT NULL,
    text        VARCHAR(700) NOT NULL,
    image_label VARCHAR(200),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_inspirations_owner_date
    ON inspirations (user_id, item_date DESC, id DESC);
