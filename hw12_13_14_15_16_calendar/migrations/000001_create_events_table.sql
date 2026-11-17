-- +goose Up
CREATE TABLE IF NOT EXISTS events (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    starts_at TIMESTAMPTZ NOT NULL,
    duration BIGINT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    user_id TEXT NOT NULL,
    notify_before BIGINT NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS events_user_starts_at_idx ON events (user_id, starts_at);

-- +goose Down
DROP TABLE IF EXISTS events;
