-- +goose Up
CREATE TABLE IF NOT EXISTS model_preference (
    id TEXT PRIMARY KEY,
    enabled INTEGER NOT NULL DEFAULT 1,
    provider_id TEXT NOT NULL,
    display_name TEXT NOT NULL DEFAULT '',
    is_custom INTEGER NOT NULL DEFAULT 0,
    time_created INTEGER NOT NULL,
    time_updated INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_model_preference_provider ON model_preference(provider_id);

-- +goose Down
DROP TABLE IF EXISTS model_preference;