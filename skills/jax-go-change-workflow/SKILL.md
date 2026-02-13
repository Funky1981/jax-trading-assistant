---
name: jax-go-change-workflow
description: Standard workflow for Go changes in jax-trading-assistant across `services/*`, `libs/*`, and shared modules. Use when implementing or reviewing backend changes that require consistent formatting, linting, package-scoped testing, and dependency-aware verification.
---

# Jax Go Change Workflow

Run a repeatable backend workflow that minimizes regressions and avoids under-testing.

## Workflow

1. Identify scope:
   - single package
   - one service
   - shared library consumed by multiple services
2. Format first, then test:
   - run `scripts/go-verify.ps1 -Mode quick -Packages ./path/...` during iteration
   - run `scripts/go-verify.ps1 -Mode standard -Packages ./path/...` before handoff
3. Expand test scope if shared packages changed:
   - include dependents from `references/test-matrix.md`
4. Escalate to full verification when changing core contracts, risk/execution logic, or cross-service glue:
   - run `scripts/go-verify.ps1 -Mode full`

## Guardrails

- Keep edits package-local unless the behavior requires cross-package changes.
- Avoid introducing hidden coupling between `services/*` and unrelated `libs/*`.
- Preserve public interfaces in `libs/*` unless a coordinated migration is part of the task.

## Reporting

- Report mode used (`quick`, `standard`, or `full`) and package targets.
- Call out skipped checks explicitly, with reason.
