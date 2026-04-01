# Jax Paper Trading Finish Plan — Phased Pack v1

This pack reorganizes the finish plan into phase folders for Codex desktop execution.

## Execute in order
1. rebaseline the pack against the current `work` branch
2. phase-01-truth-path-hardening
3. phase-02-data-and-strategy-model
4. phase-03-always-on-watcher-and-candidates
5. phase-04-approval-queue-and-paper-execution
6. phase-06-assistant-and-ai-audit
7. phase-07-trust-gates-and-signoff
8. treat phase-05-operator-pages as mostly landed and only patch contract drift where needed

## Cross-cutting controls
Use these in every phase:
- `cross-cutting-controls/12_DB_SCHEMA_AND_MIGRATIONS_PLAN.md`
- `cross-cutting-controls/13_API_COMPLETION_PLAN.md`
- `cross-cutting-controls/15_CODEX_EXECUTION_ORDER.md`
- `cross-cutting-controls/16_NEXT_10_HIGHEST_VALUE_TASKS.md`
- `cross-cutting-controls/17_SCOREBOARD_TEMPLATE_PAPER_READY.md`
- `cross-cutting-controls/18_CODEX_MASTER_PROMPT.md`

## Rules
- no fake/synthetic data in research or paper truth paths
- AI is advisory only
- no approval bypass
- no gate pass, no paper-trading sign-off

## Current branch reality
Already present on `work`:
- strategy-instance CRUD
- `/research`, `/analysis`, `/testing`, `/approvals`, `/assistant`
- candidate / approval / chat schemas
- AI decision tables
- trust-gate endpoints

Remaining work is hardening, linkage, and proof generation.
