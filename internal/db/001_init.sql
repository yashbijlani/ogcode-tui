-- +goose Up
CREATE TABLE IF NOT EXISTS session (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    directory TEXT NOT NULL,
    title TEXT NOT NULL DEFAULT '',
    permission TEXT,
    time_created INTEGER NOT NULL,
    time_updated INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_session_project ON session(project_id);

CREATE TABLE IF NOT EXISTS message (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES session(id) ON DELETE CASCADE,
    data TEXT NOT NULL,
    time_created INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_message_session ON message(session_id, time_created);

CREATE TABLE IF NOT EXISTS part (
    id TEXT PRIMARY KEY,
    message_id TEXT NOT NULL REFERENCES message(id) ON DELETE CASCADE,
    session_id TEXT NOT NULL,
    data TEXT NOT NULL,
    time_created INTEGER NOT NULL,
    time_updated INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_part_message ON part(message_id);
CREATE INDEX IF NOT EXISTS idx_part_session ON part(session_id);

CREATE TABLE IF NOT EXISTS permission_request (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    tool TEXT NOT NULL,
    input TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    response TEXT,
    time_created INTEGER NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS part;
DROP TABLE IF EXISTS message;
DROP TABLE IF EXISTS session;
DROP TABLE IF EXISTS permission_request;