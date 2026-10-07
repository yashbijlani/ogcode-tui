-- +goose Up
ALTER TABLE session ADD COLUMN memory_tokens_saved INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE session DROP COLUMN memory_tokens_saved;
