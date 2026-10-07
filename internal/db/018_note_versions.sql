-- +goose Up
ALTER TABLE note ADD COLUMN version INTEGER NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS note_version (
    id TEXT PRIMARY KEY,
    note_id TEXT NOT NULL REFERENCES note(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    content TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_note_version_note ON note_version(note_id);

-- +goose Down
DROP TABLE IF EXISTS note_version;
