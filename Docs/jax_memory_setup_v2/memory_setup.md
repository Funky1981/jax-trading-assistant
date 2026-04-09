# Jax Memory System (Phase 1 - Hybrid)

## Goal
Postgres = truth  
pgvector = semantic search  
No external memory frameworks

# Enable pgvector
CREATE EXTENSION IF NOT EXISTS vector;

# Tables

-- Research
CREATE TABLE research (
    id BIGSERIAL PRIMARY KEY,
    symbol TEXT NOT NULL,
    summary TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    embedding VECTOR(1536)
);

-- Trades
CREATE TABLE trades (
    id BIGSERIAL PRIMARY KEY,
    symbol TEXT NOT NULL,
    entry_price NUMERIC,
    stop_price NUMERIC,
    outcome TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Events
CREATE TABLE events (
    id BIGSERIAL PRIMARY KEY,
    symbol TEXT NOT NULL,
    type TEXT,
    data JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);

# Index
CREATE INDEX research_embedding_idx
ON research
USING hnsw (embedding vector_cosine_ops);

# Query Example
SELECT *
FROM research
WHERE symbol = 'AAPL'
ORDER BY embedding <=> '[query_vector]'
LIMIT 5;
