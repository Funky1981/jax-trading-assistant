# Skills Quick Guide

If skills are new to you, use this file as your day-to-day cheat sheet.

## Quick Start

Use one of these at the start of your prompt:

- `Use skills/STARTER_PROMPT.md routing for this task.`
- `Use jax-repo-guardrails and jax-go-change-workflow for this.`
- `Use jax-orchestration-pipeline and jax-debug-runbook for this issue.`

You can also just describe your task normally. The agent can auto-select skills from descriptions.

## When to Invoke Each Skill

- `jax-repo-guardrails`
  - Use when unsure where to edit, what to avoid, or how wide validation should be.
  - Good default for almost every code task.

- `jax-go-change-workflow`
  - Use for backend Go code in `services/*`, `libs/*`, `internal/*`.
  - Covers formatting, linting, and scoped test flow.

- `jax-golden-replay-regression`
  - Use when behavior might change: signals, execution, orchestration outputs.
  - Adds golden/replay verification discipline.

- `jax-orchestration-pipeline`
  - Use for orchestration request/response flow issues across API, orchestrator, signal generator, and provider config.

- `jax-frontend-api-contracts`
  - Use when frontend data/hooks/pages depend on backend contract changes.

- `jax-adr0012-migration`
  - Use for modular-monolith migration work tied to ADR-0012 phases.

- `jax-debug-runbook`
  - Use when things fail to start, health checks fail, or runtime behavior is broken.

- `jax-knowledge-ingest`
  - Use for `knowledge/md` updates, schema apply, and ingest runs.

## Common Combinations

1. Backend feature/fix:
   - `jax-repo-guardrails` + `jax-go-change-workflow`

2. Backend change with possible behavior drift:
   - `jax-repo-guardrails` + `jax-go-change-workflow` + `jax-golden-replay-regression`

3. Orchestration bug:
   - `jax-repo-guardrails` + `jax-orchestration-pipeline` + `jax-debug-runbook`

4. Frontend + backend contract update:
   - `jax-repo-guardrails` + `jax-go-change-workflow` + `jax-frontend-api-contracts`

5. ADR-0012 migration ticket:
   - `jax-repo-guardrails` + `jax-adr0012-migration` (+ `jax-golden-replay-regression` if behavior-sensitive)

## Copy/Paste Prompts

- `Use jax-repo-guardrails + jax-go-change-workflow. Fix <bug>. Run minimal valid tests and report what you ran.`
- `Use jax-orchestration-pipeline + jax-debug-runbook. Trace why <endpoint/flow> fails and propose a minimal fix.`
- `Use jax-golden-replay-regression. Verify whether this change alters behavior, and only refresh baseline if intentional.`
- `Use jax-knowledge-ingest. Run dry-run ingest first, then full ingest if clean.`

## Rule of Thumb

If you are unsure, start with:

`Use skills/STARTER_PROMPT.md routing for this task.`
