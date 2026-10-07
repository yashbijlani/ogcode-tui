-- +goose Up
ALTER TABLE search_config ADD COLUMN use_real_profile INTEGER NOT NULL DEFAULT 0;

-- +goose Down
-- SQLite does not support DROP COLUMN on older versions; leave the column in place.
