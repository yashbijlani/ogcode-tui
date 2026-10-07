-- +goose Up
ALTER TABLE session ADD COLUMN model TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE session DROP COLUMN model;