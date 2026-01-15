-- Entries: the captured content
CREATE TABLE IF NOT EXISTS entries (
    id TEXT PRIMARY KEY,
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_viewed_at TIMESTAMP
);

-- Tags: emergent from classification
CREATE TABLE IF NOT EXISTS tags (
    id TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    parent_id TEXT REFERENCES tags(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Entry-Tag relations (many-to-many)
CREATE TABLE IF NOT EXISTS entry_tags (
    entry_id TEXT REFERENCES entries(id) ON DELETE CASCADE,
    tag_id TEXT REFERENCES tags(id) ON DELETE CASCADE,
    confidence REAL DEFAULT 1.0,
    PRIMARY KEY (entry_id, tag_id)
);

-- Index for faster tag lookups
CREATE INDEX IF NOT EXISTS idx_entry_tags_entry ON entry_tags(entry_id);
CREATE INDEX IF NOT EXISTS idx_entry_tags_tag ON entry_tags(tag_id);
CREATE INDEX IF NOT EXISTS idx_tags_parent ON tags(parent_id);

-- Embeddings for similarity search
CREATE TABLE IF NOT EXISTS embeddings (
    entry_id TEXT PRIMARY KEY REFERENCES entries(id) ON DELETE CASCADE,
    vector BLOB NOT NULL,
    model TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
