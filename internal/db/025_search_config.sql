-- +goose Up
CREATE TABLE IF NOT EXISTS search_config (
    id           INTEGER PRIMARY KEY CHECK (id = 1),
    enabled      INTEGER NOT NULL DEFAULT 0,
    time_updated INTEGER NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS search_config;
