-- +goose Up
ALTER TABLE session ADD COLUMN compaction_summary TEXT NOT NULL DEFAULT '';

-- +goose Down
SELECT 1; -- SQLite does not support DROP COLUMN in older versions
