-- +goose Up
CREATE TABLE IF NOT EXISTS provider_config (
    provider_id TEXT PRIMARY KEY,
    api_key     TEXT NOT NULL DEFAULT '',
    base_url    TEXT NOT NULL DEFAULT '',
    time_updated INTEGER NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS provider_config;
