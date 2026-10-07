-- +goose Up
CREATE TABLE IF NOT EXISTS theme (
    directory TEXT PRIMARY KEY,
    primary_color TEXT NOT NULL,
    accent TEXT NOT NULL,
    accent_hover TEXT NOT NULL,
    accent_soft TEXT NOT NULL,
    accent_ring TEXT NOT NULL,
    on_primary TEXT NOT NULL,
    glow TEXT NOT NULL,
    tint TEXT NOT NULL DEFAULT 'rgba(59, 130, 246, 0.05)',
    updated_at INTEGER NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS theme;