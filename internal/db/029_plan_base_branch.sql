-- +goose Up
-- Branch that task PRs from this plan should target (the repo's active branch,
-- captured when the plan is locked). Empty means fall back to the default branch.
ALTER TABLE plan ADD COLUMN base_branch TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE plan DROP COLUMN base_branch;
