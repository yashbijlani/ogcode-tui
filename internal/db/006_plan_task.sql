-- +goose Up
-- Plan table: metadata for a planning-mode session
-- Note: we also add session_type to the session table in a separate migration
-- to keep migrations atomic. This migration only creates plan + task tables.

CREATE TABLE IF NOT EXISTS plan (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES session(id) ON DELETE CASCADE,
    project_id TEXT NOT NULL,
    directory TEXT NOT NULL,
    title TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'open',
    model TEXT NOT NULL DEFAULT '',
    compaction_summary TEXT NOT NULL DEFAULT '',
    time_created INTEGER NOT NULL,
    time_updated INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_plan_session ON plan(session_id);
CREATE INDEX IF NOT EXISTS idx_plan_project ON plan(project_id);
CREATE INDEX IF NOT EXISTS idx_plan_status ON plan(status);

-- Task table: each task is derived from a locked plan
CREATE TABLE IF NOT EXISTS task (
    id TEXT PRIMARY KEY,
    plan_id TEXT NOT NULL REFERENCES plan(id) ON DELETE CASCADE,
    session_id TEXT REFERENCES session(id) ON DELETE SET NULL,
    parent_task_id TEXT REFERENCES task(id) ON DELETE SET NULL,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    effort TEXT NOT NULL DEFAULT 'M',
    complexity TEXT NOT NULL DEFAULT 'medium',
    status TEXT NOT NULL DEFAULT 'pending',
    dependencies TEXT NOT NULL DEFAULT '[]',
    branch_name TEXT NOT NULL DEFAULT '',
    pr_url TEXT NOT NULL DEFAULT '',
    pr_number INTEGER,
    order_index INTEGER NOT NULL DEFAULT 0,
    time_created INTEGER NOT NULL,
    time_updated INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_task_plan ON task(plan_id);
CREATE INDEX IF NOT EXISTS idx_task_status ON task(status);
CREATE INDEX IF NOT EXISTS idx_task_session ON task(session_id);

-- +goose Down
DROP TABLE IF EXISTS task;
DROP TABLE IF EXISTS plan;