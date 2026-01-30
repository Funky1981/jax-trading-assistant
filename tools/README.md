# Jax Knowledge Ingestion (Go + PostgreSQL)

Generated: 2026-01-30

This package provides a **MD-first, production-friendly ingestion pipeline** for Jax using **Go** and **PostgreSQL**.

## Why MD + DB (hybrid)
- **Markdown in Git** = source of truth (reviewable, versioned, auditable)
- **PostgreSQL** = queryable registry + fast runtime access
- Optional: **vector store** (Qdrant) for semantic retrieval later

## What you get
- `sql/schema.sql` — PostgreSQL schema (documents + optional registry)
- `cmd/ingest/` — Go CLI ingestor
- `docker-compose.yml` — Postgres (+ optional Qdrant via profile)
- `docs/` — security + observability guidance

## Run (local)
1) Start dependencies:
```bash
docker compose up -d
```

2) Apply schema (one-time):
```bash
psql "postgres://postgres:postgres@localhost:5432/jax_knowledge?sslmode=disable" -f sql/schema.sql
```

3) Ingest a knowledge base folder (the MD zip you downloaded earlier):
```bash
go run ./cmd/ingest \
  --root ../jax_strategy_knowledge_base \
  --dsn "postgres://postgres:postgres@localhost:5432/jax_knowledge?sslmode=disable" \
  --dry-run=false
```

## Contract: YAML front matter
Docs should start with:
```yaml
---
title: "…"
version: "1.0"
status: "approved"   # approved|candidate|retired|draft
created_utc: "2026-01-30"
tags: ["strategy","trend"]
---
```

## Safety rule
At runtime, Jax must only treat `status = 'approved'` as eligible for live trading.
