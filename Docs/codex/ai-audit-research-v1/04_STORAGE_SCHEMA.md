# Storage schema (save everything for analysis)

You already have `audit_events`. Extend with dedicated tables for AI and runs.

## Tables to add (minimum)
1) `ai_decisions`
2) `ai_decision_acceptance`
3) `runs`
4) `run_artifacts`
5) `test_runs`
6) `gate_status`

Optional but recommended:
- `order_intents` (separate from trades)
- `fills` (separate table per fill)
- `reconciliations` (daily summaries)

## Relationship model
- `runs` is parent
- `ai_decisions` links to `runs` and `instance_id`
- `trades` links to `runs` and `instance_id`
- `order_intents` links to `signals` and `ai_decisions` (if AI contributed)
- `fills` link to `trades`
- `gate_status` links to latest `test_runs`

## Retention
- Do not delete AI decisions for runs you care about.
- If storage is a concern, store raw text compressed or in object storage and keep references.

