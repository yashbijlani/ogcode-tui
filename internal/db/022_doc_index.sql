-- +goose Up
CREATE TABLE IF NOT EXISTS doc_page_index (
    id TEXT PRIMARY KEY,
    doc_path TEXT NOT NULL,
    page_num INTEGER NOT NULL,
    keywords TEXT NOT NULL DEFAULT '[]',
    labels TEXT NOT NULL DEFAULT '[]',
    indexed_at INTEGER NOT NULL,
    UNIQUE(doc_path, page_num)
);
CREATE INDEX IF NOT EXISTS idx_doc_page_index_path ON doc_page_index(doc_path);

-- +goose Down
DROP TABLE IF EXISTS doc_page_index;
