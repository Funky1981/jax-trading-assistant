-- Enable pgvector extension required for embedding storage and cosine search.
CREATE EXTENSION IF NOT EXISTS vector;

-- memory_items: unified source-of-truth for all memory banks.
CREATE TABLE IF NOT EXISTS memory_items (
    id            TEXT        NOT NULL DEFAULT gen_random_uuid()::text,
    bank          TEXT        NOT NULL,
    ts            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    type          TEXT        NOT NULL,
    symbol        TEXT,
    summary       TEXT        NOT NULL,
    tags          JSONB       NOT NULL DEFAULT '[]',
    data          JSONB,
    source_system TEXT,
    source_ref    TEXT,
    embedding     vector(1536),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id)
);

-- Retrieval by bank ordered by recency.
CREATE INDEX IF NOT EXISTS idx_memory_items_bank_ts
    ON memory_items (bank, ts DESC);

-- Symbol-scoped retrieval within a bank.
CREATE INDEX IF NOT EXISTS idx_memory_items_bank_symbol_ts
    ON memory_items (bank, symbol, ts DESC);

-- GIN index for JSONB tag containment queries.
CREATE INDEX IF NOT EXISTS idx_memory_items_tags
    ON memory_items USING gin (tags);

-- HNSW index for approximate cosine nearest-neighbour search on embeddings.
CREATE INDEX IF NOT EXISTS idx_memory_items_embedding_hnsw
    ON memory_items USING hnsw (embedding vector_cosine_ops);
