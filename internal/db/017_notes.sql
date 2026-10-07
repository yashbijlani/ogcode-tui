-- +goose Up
CREATE TABLE IF NOT EXISTS note (
    id TEXT PRIMARY KEY,
    directory TEXT NOT NULL,
    title TEXT NOT NULL DEFAULT '',
    query TEXT NOT NULL DEFAULT '',
    content TEXT NOT NULL DEFAULT '',
    session_id TEXT,
    status TEXT NOT NULL DEFAULT 'generating',
    time_created INTEGER NOT NULL,
    time_updated INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_note_directory ON note(directory);
CREATE INDEX IF NOT EXISTS idx_note_session ON note(session_id);

-- +goose Down
DROP TABLE IF EXISTS note;
