-- +goose Up
-- Add worktree_path column to task table for parallel task execution via git worktrees

ALTER TABLE task ADD COLUMN worktree_path TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE task DROP COLUMN worktree_path;