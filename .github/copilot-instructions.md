# Copilot Instructions for jax-trading-assistant

Use these instructions as the default behavior for all coding tasks in this repository.

## Core Behavior

- Choose the minimum scope and minimum file set needed.
- Prefer edits in first-party code; avoid vendored code unless explicitly requested.
- Explain planned checks, run the smallest valid verification set, and report what was run/skipped.
- Keep behavior stable unless the task explicitly asks for behavior change.

## Repository Boundaries

- First-party areas:
  - `services/`, `libs/`, `internal/`, `frontend/`, `tests/`, `tools/`, `db/`, `config/`, `Docs/`
- Treat as vendored/external unless explicitly requested:
  - `services/hindsight/`, `Agent0/`, `dexter/`

## Skill-to-Task Routing (Use as Playbooks)

When handling a task, follow the matching playbook(s):

- Scope/safety/pathing:
  - `skills/jax-repo-guardrails/SKILL.md`
- Go backend code:
  - `skills/jax-go-change-workflow/SKILL.md`
- Behavior-sensitive logic (signals/execution/orchestration):
  - `skills/jax-golden-replay-regression/SKILL.md`
- Orchestration flow/contracts:
  - `skills/jax-orchestration-pipeline/SKILL.md`
- Frontend/backend contract changes:
  - `skills/jax-frontend-api-contracts/SKILL.md`
- ADR-0012 migration work:
  - `skills/jax-adr0012-migration/SKILL.md`
- Runtime debugging/startup/health:
  - `skills/jax-debug-runbook/SKILL.md`
- Knowledge DB/schema/ingest:
  - `skills/jax-knowledge-ingest/SKILL.md`

Default multi-skill order:
1. `jax-repo-guardrails`
2. Domain playbook
3. `jax-golden-replay-regression` (only if behavior-sensitive)

## Verification Defaults

- Go changes:
  - `skills/jax-go-change-workflow/scripts/go-verify.ps1`
- Golden/replay checks:
  - `skills/jax-golden-replay-regression/scripts/golden-check.ps1`
- Knowledge ingest cycle:
  - `skills/jax-knowledge-ingest/scripts/knowledge-cycle.ps1`

## ADR-0012 Guardrail

For modular-monolith migration tasks, align to phase gates in:
- `Docs/ADR-0012-two-runtime-modular-monolith.md`
- `skills/jax-adr0012-migration/references/phase-checklist.md`

Do not claim a phase complete without matching evidence (tests and boundary checks).
