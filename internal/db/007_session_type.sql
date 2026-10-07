-- +goose Up
ALTER TABLE session ADD COLUMN session_type TEXT NOT NULL DEFAULT 'build';
-- 'build' = regular coding session, 'plan' = planning session

-- +goose Down
SELECT 1; -- SQLite does not support DROP COLUMN in older versions