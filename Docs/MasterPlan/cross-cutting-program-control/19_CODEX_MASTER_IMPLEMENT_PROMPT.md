# Codex Master Implement Prompt

Use on `work` branch.

Goal:
Implement the full robust Jax event-trading platform plan for IBM-style news repricing events, including no-fake-data enforcement, event ingestion, strategy framework, research truth path, execution hardening, AI audit/replay, UI operations pages, trust gates, and shadow validation.

Rules:
- No fake/synthetic data in paper/live paths.
- Deterministic code authoritative for strategy/risk/orders/P&L/gates.
- AI advisory only, schema-validated, logged, replayable.
- Every phase includes tests, migrations, docs, and audit hooks.

Strict implementation order:
1) No-fake-data enforcement and provenance
2) Event + intraday data correctness
3) Strategy types + instances
4) Research truth path + reproducibility
5) Execution hardening + flatten/reconciliation
6) AI audit + decision trace
7) UI route wiring + Research/Analysis/Testing integration
8) Trust gates + shadow validation
9) Extended paper trial tooling

Start with Phase 1:
Audit all fake/stub/demo generators and produce a disposition list (remove/dev-only/test-only), then implement runtime/build gating and provenance fields.
