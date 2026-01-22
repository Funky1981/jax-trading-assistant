# Jax Trading Assistant - Plan (Agent0 + go-UTCP + Dexter + Hindsight) with TDD

This folder contains the build plan docs for Codex.

**Important:** You said Codex is currently implementing **02**.  
So the biggest change is: **from 03 onwards**, we introduce **Hindsight** as the memory layer, and we require **test-driven development** for every new module and every tool endpoint. We also make sure *every tested/verified outcome can be logged into memory* (where appropriate).

## Quick map

- 01-02: repo + baseline scaffolding (keep aligned with what you already started)
- 03-05: add Hindsight + memory schemas + UTCP "memory tools"
- 06-07: wire Agent0 + Dexter into the memory pipeline
- 08: reflection jobs (the "learning loop")
- 09: observability (logs/metrics/traces) + what gets retained
- 10: TDD strategy + test taxonomy + fixtures
- 11: docker-compose for local dev
- 12: CI pipeline guidance

## Folder structure inside this zip

- `backend/` - numbered step-by-step build documents (what Codex follows)
- `frontend/` - UI documentation (architecture, components, styling, testing)
- `.serena/templates/` - copy/paste snippets (schemas, example payloads)
- `.serena/checklists/` - "definition of done" for each step
