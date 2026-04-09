CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS research (
    id BIGSERIAL PRIMARY KEY,
    symbol TEXT NOT NULL,
    summary TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    embedding VECTOR(1536)
);

CREATE TABLE IF NOT EXISTS trades (
    id BIGSERIAL PRIMARY KEY,
    symbol TEXT NOT NULL,
    entry_price NUMERIC,
    stop_price NUMERIC,
    outcome TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS events (
    id BIGSERIAL PRIMARY KEY,
    symbol TEXT NOT NULL,
    type TEXT,
    data JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS research_embedding_idx
ON research
USING hnsw (embedding vector_cosine_ops);
