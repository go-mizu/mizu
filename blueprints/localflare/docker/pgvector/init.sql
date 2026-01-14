-- Enable pgvector extension
CREATE EXTENSION IF NOT EXISTS vector;

-- Create indexes table
CREATE TABLE IF NOT EXISTS vector_indexes (
    name TEXT PRIMARY KEY,
    dimensions INTEGER NOT NULL,
    metric TEXT NOT NULL DEFAULT 'cosine',
    description TEXT,
    vector_count BIGINT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create vectors table (partitioned by index_name for better performance)
CREATE TABLE IF NOT EXISTS vectors (
    id TEXT NOT NULL,
    index_name TEXT NOT NULL REFERENCES vector_indexes(name) ON DELETE CASCADE,
    namespace TEXT DEFAULT '',
    embedding vector,
    metadata JSONB DEFAULT '{}',
    PRIMARY KEY (index_name, id)
);

-- Create index on namespace for filtered queries
CREATE INDEX IF NOT EXISTS idx_vectors_namespace ON vectors(index_name, namespace);
