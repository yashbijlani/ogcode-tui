-- +goose Up
ALTER TABLE task ADD COLUMN pr_error TEXT NOT NULL DEFAULT '';

-- +goose Down
-- SQLite does not support DROP COLUMN in older versions, so this is a no-op.
