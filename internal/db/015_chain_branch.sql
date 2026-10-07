-- +goose Up
ALTER TABLE task ADD COLUMN chain_branch TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE task DROP COLUMN chain_branch;
