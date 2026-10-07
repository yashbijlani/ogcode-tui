-- +goose Up
CREATE TABLE IF NOT EXISTS index_excludes (
    id TEXT PRIMARY KEY,
    directory TEXT NOT NULL,
    pattern TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    UNIQUE(directory, pattern)
);
CREATE INDEX IF NOT EXISTS idx_index_excludes_dir ON index_excludes(directory);

-- +goose Down
DROP TABLE IF EXISTS index_excludes;
