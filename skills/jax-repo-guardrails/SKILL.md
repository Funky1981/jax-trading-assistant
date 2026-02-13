---
name: jax-repo-guardrails
description: Repository routing, safety boundaries, and validation guardrails for jax-trading-assistant. Use when planning edits, choosing where code should live, deciding what not to touch, selecting command/test scope, or reviewing change impact across services, libs, frontend, docs, and vendored components.
---

# Jax Repo Guardrails

Map a request to the correct area of this monorepo and avoid cross-boundary mistakes before editing.

## Workflow

1. Classify the request into one primary area: backend service, shared `libs/`, frontend, infra/config, testing, or docs.
2. Use `references/path-map.md` to select the narrowest directory and likely validation commands.
3. Confirm whether the task touches runtime-critical boundaries:
   - trading execution and risk paths
   - orchestration API contracts
   - UTCP contracts
   - knowledge ingestion schema
4. Refuse speculative broad edits. Keep changes local to the minimum set of files needed for behavior.
5. Validate with the smallest command set that still protects correctness.

## Hard Boundaries

- Treat `services/hindsight/`, `Agent0/`, and `dexter/` as vendored/external unless explicitly requested.
- Avoid editing generated/build artifacts such as `*.exe`, lockfiles, and archived docs unless task-specific.
- Preserve architecture intent documented in `Docs/ARCHITECTURE.md` and ADRs in `Docs/`.

## Validation Selection

- For Go backend edits: run `gofmt`, package-level tests, then broader tests if shared libs changed.
- For frontend edits: run targeted `vitest` and affected `e2e` specs where applicable.
- For behavior-sensitive trading/orchestration edits: run golden/replay checks.

## Output Expectations

- State why each touched file is in scope.
- Call out untouched but adjacent risky areas.
- Report exactly which commands were run and which were skipped.
