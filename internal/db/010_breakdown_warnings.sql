-- +goose Up
ALTER TABLE plan ADD COLUMN breakdown_warnings TEXT NOT NULL DEFAULT '';

-- +goose Down
-- SQLite does not support DROP COLUMN in older versions, so this is a no-op.
