-- +goose Up
CREATE TABLE IF NOT EXISTS memory_config (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    enabled INTEGER NOT NULL DEFAULT 0,
    embed_provider_id TEXT NOT NULL DEFAULT '',
    embed_model TEXT NOT NULL DEFAULT '',
    embed_api_key TEXT NOT NULL DEFAULT '',
    chat_provider_id TEXT NOT NULL DEFAULT '',
    chat_model TEXT NOT NULL DEFAULT '',
    chat_api_key TEXT NOT NULL DEFAULT '',
    time_updated INTEGER NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS memory_config;
