# AGENTS.md instructions for c:\Projects\jax-trading assistant

## Skills

A skill is a set of local instructions to follow that is stored in a `SKILL.md` file.

### Available skills

- jax-repo-guardrails: Repository routing, safety boundaries, and validation scope selection for this monorepo. (file: skills/jax-repo-guardrails/SKILL.md)
- jax-go-change-workflow: Standard Go implementation and verification workflow for `services/*` and `libs/*`. (file: skills/jax-go-change-workflow/SKILL.md)
- jax-golden-replay-regression: Deterministic regression workflow using golden snapshots and replay tests. (file: skills/jax-golden-replay-regression/SKILL.md)
- jax-orchestration-pipeline: End-to-end orchestration tracing and contract-preserving edit workflow. (file: skills/jax-orchestration-pipeline/SKILL.md)
- jax-frontend-api-contracts: Frontend/backend contract workflow for `frontend/src/data`, hooks, and API-facing pages. (file: skills/jax-frontend-api-contracts/SKILL.md)
- jax-adr0012-migration: Phase-aligned migration guardrails for ADR-0012 modular-monolith work. (file: skills/jax-adr0012-migration/SKILL.md)
- jax-debug-runbook: Service and environment triage workflow for startup/health/runtime failures. (file: skills/jax-debug-runbook/SKILL.md)
- jax-knowledge-ingest: Knowledge DB schema and ingest workflow for `knowledge/md` and `tools/cmd/ingest`. (file: skills/jax-knowledge-ingest/SKILL.md)

### How to use skills

- Trigger a skill by naming it (for example, `jax-go-change-workflow`) or by asking for work that clearly matches its description.
- Use only the minimum set of skills needed for the request.
- Open the listed `SKILL.md` and read only the parts needed to perform the task.
- Load files from each skill's `references/` or `scripts/` folders only when needed.
- For quick routing, start with `skills/STARTER_PROMPT.md`.

## Default behavior

- In this repository, automatically select and use the appropriate skill(s) for every task unless the user explicitly asks not to use skills.
- Announce chosen skill(s) in one short line before doing substantive work.
