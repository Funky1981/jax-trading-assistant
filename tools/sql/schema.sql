-- Jax Knowledge Base schema (PostgreSQL)
-- Purpose: store strategy docs with versioning + structured fields for governance

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS strategy_documents (
  doc_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  doc_type TEXT NOT NULL,          -- strategy|pattern|anti-pattern|meta|risk|evaluation|unknown
  rel_path TEXT NOT NULL UNIQUE,   -- repo-relative path
  title TEXT NOT NULL,
  version TEXT NOT NULL,
  status TEXT NOT NULL,            -- approved|candidate|retired|draft
  created_utc TIMESTAMPTZ NOT NULL,
  updated_utc TIMESTAMPTZ NOT NULL,
  tags JSONB NOT NULL DEFAULT '[]'::jsonb,
  sha256 TEXT NOT NULL,
  markdown TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS ix_strategy_documents_doc_type ON strategy_documents(doc_type);
CREATE INDEX IF NOT EXISTS ix_strategy_documents_status ON strategy_documents(status);
CREATE INDEX IF NOT EXISTS ix_strategy_documents_tags_gin ON strategy_documents USING gin(tags);

-- Optional registry table if you want stable keys separate from file paths
CREATE TABLE IF NOT EXISTS strategy_registry (
  strategy_key TEXT PRIMARY KEY,
  current_doc_id UUID NULL REFERENCES strategy_documents(doc_id),
  status TEXT NOT NULL,
  updated_utc TIMESTAMPTZ NOT NULL
);
