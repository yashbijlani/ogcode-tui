-- +goose Up
CREATE TABLE IF NOT EXISTS model_capability (
    model_id        TEXT PRIMARY KEY,
    supports_images INTEGER NOT NULL DEFAULT 0,
    probed_at       INTEGER NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS model_capability;
