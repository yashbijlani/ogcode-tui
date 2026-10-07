-- +goose Up
ALTER TABLE memory_config ADD COLUMN embed_base_url TEXT NOT NULL DEFAULT '';
ALTER TABLE memory_config ADD COLUMN chat_base_url TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE memory_config DROP COLUMN chat_base_url;
ALTER TABLE memory_config DROP COLUMN embed_base_url;