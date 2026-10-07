-- +goose Up
ALTER TABLE note ADD COLUMN source TEXT NOT NULL DEFAULT 'ai';

-- +goose Down
ALTER TABLE note DROP COLUMN source;
