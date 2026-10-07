-- +goose Up
-- Per-task model override. Empty means inherit the plan's model.
ALTER TABLE task ADD COLUMN model TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE task DROP COLUMN model;
