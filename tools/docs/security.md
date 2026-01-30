# Security & Governance (Go + Postgres)

## Non-negotiables
- Markdown in Git is the **source of truth** (PRs decide).
- PostgreSQL is a derived index/cache (can be rebuilt anytime).
- Only `status='approved'` docs are eligible for live trading.

## Controls
- CI validation of front matter schema
- Role-based DB access:
  - Ingestor: write access
  - Runtime Jax: read-only access (recommended)
- Optional signing/tagging of approved releases
