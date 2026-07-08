# Jax Roadmap Reconciliation Report

## A. Current Repo Status Summary

New roadmap located at:

- `Docs/plans/jax-trading-roadmap-pack/docs/roadmap/ROADMAP.md`
- `Docs/plans/jax-trading-roadmap-pack/docs/roadmap/PHASE_GATES.md`
- `Docs/plans/jax-trading-roadmap-pack/docs/risk/RISK_AND_SLIPPAGE_RULES.md`
- `Docs/plans/jax-trading-roadmap-pack/docs/templates/TRADE_CANDIDATE_TEMPLATE.md`
- `Docs/plans/jax-trading-roadmap-pack/docs/templates/TRADE_REVIEW_TEMPLATE.md`

Note: the request mentioned `Docs/plans/roadmap/`, but the actual new path is `Docs/plans/jax-trading-roadmap-pack/docs/roadmap/`.

### What Exists In Code Now

- Deterministic decisioning pipeline: event classification, decision core, swing brain, risk veto, research evidence validation, paper ticket creation, review scheduling.
- Candidate trade persistence and lifecycle: `candidate_trades`, `candidate_events`, candidate status transitions, duplicate guard, expiry.
- Human approval flow: approval queue, approve/reject/snooze/reanalyze, mobile approval token flow, Telegram notification outbox.
- Paper execution instruction path: approved candidates can create `execution_instructions`; a paper-mode worker can submit through the execution service when safety env flags allow it.
- ETF paper guardrails: allowlist, blocked leveraged/inverse/volatility products, paper-only mode checks, quote freshness, spread, regular trading hours, stop-loss, flatten-by-close.
- Review/journal foundations: decision logs, outcome reviews, lessons, review schedules, persisted review operation/read-model workflows.
- Runtime controls: `JAX_RUNTIME_MODE`, `ALLOW_LIVE_TRADING`, `IB_PAPER_TRADING`, provider policy checks, `global_kill_switch` check for candidate scanning.

### What Exists Only Or Mostly In Docs

- The new roadmap's full structured candidate shape is not fully represented in the main candidate model. Missing or partial fields include catalyst, evidence item list, market regime, slippage allowance, max normal loss, max slippage-adjusted loss, and invalidation as first-class fields.
- Evidence scoring exists as numeric decision scores, but not as the new roadmap's structured evidence items with source, timestamp, confidence, impact, and contradictory evidence.
- Slippage-aware risk sizing exists in docs and schema, but the active execution sizing still uses `abs(entry-stop)` without slippage buffer.
- Journal intelligence is partially implemented via review/lesson types and `trade_reviews` schema, but not as the new review template end-to-end.
- Shadow mode has a `cmd/shadow-validator` entry point and old docs, but there is not enough evidence to treat full roadmap Phase 7 as implemented.

### What Appears Stale Or Duplicated

- `Docs/ROADMAP.md` is stale relative to the new roadmap. It says live trading is `NOT_PLANNED`; the new roadmap allows tiny live activation only after explicit late gates.
- Archived autonomous plans contain automatic execution language and should not guide current work.
- Multiple archived phase packs duplicate candidate/approval/risk/trust-gate plans. Keep them as historical references only.
- `config/risk-constraints.json` previously allowed leverage above the safety baseline; it now requires `max_leverage <= 1.0`.

## B. Old Plan Reconciliation Table

| File / phase / item | Classification | Reason | Recommended action |
|---|---|---|---|
| `Docs/ROADMAP.md` active roadmap | MERGE INTO NEW ROADMAP | Useful implemented-status spine, but conflicts on live trading being not planned. | Update to state new roadmap pack is controlling; mark live only as late explicit-gate phase. |
| `docs/PHASE_CONTRACTS/00-07` | MERGE INTO NEW ROADMAP | Decision core, risk veto, approval, review align with phases 0-7. | Keep as implementation evidence; reconcile naming/status with new gates. |
| `Docs/TRADING_BRAIN/*` | MERGE INTO NEW ROADMAP | Good decision/evidence/risk/approval concepts, but less complete than new roadmap. | Treat as supporting specs, not controlling plan. |
| Archived `AUTONOMOUS_TRADING_ROADMAP.md` | SUPERSEDED | Promotes autonomous monitoring and automatic approved-trade execution. | Mark historical/non-canonical; do not implement from it. |
| Archived masterplan phase 00/01 no-fake-data, baseline hardening | KEEP | Strongly matches safety baseline. | Pull into Phase 0 checklist. |
| Archived masterplan phase 02 event data foundation | MERGE INTO NEW ROADMAP | Supports market/news/event data. | Use after safety baseline and candidate model. |
| Archived masterplan phase 07 execution/risk/flatten | PARK FOR LATER | Useful for broker reconciliation, but too execution-heavy for early phases. | Defer to new Phases 8-9 only. |
| Archived masterplan phase 10 shadow validation | MERGE INTO NEW ROADMAP | Matches new Shadow Mode. | Reuse after paper loop and review data exist. |
| Archived masterplan phase 11 controlled live readiness | MERGE INTO NEW ROADMAP | Compatible only as late tiny-live readiness. | Keep gated; no current implementation. |
| Paper finish plan phase 01 truth-path hardening | KEEP | Safety and provenance fit Phase 0. | Use now. |
| Paper finish plan phase 02 data/strategy model | MERGE INTO NEW ROADMAP | Candidate/data modeling overlaps Phase 1. | Reconcile fields with new candidate template. |
| Paper finish plan phase 03 watcher/candidates | PARK FOR LATER | Watcher exists, but candidate completeness must be fixed first. | Do not expand watcher until model/gates are stricter. |
| Paper finish plan phase 04 approval/paper execution | MERGE INTO NEW ROADMAP | Approval loop is useful; execution pieces need caution. | Keep approval; paper execution stays disabled unless explicit paper gates pass. |
| ETF trading readiness plan | KEEP | Strong paper-only ETF guardrails and leverage/inverse exclusions. | Keep, but align risk defaults with beginner-safe roadmap. |
| Robust profitability: position sizing/risk | MERGE INTO NEW ROADMAP | Directly supports risk engine and slippage-aware sizing. | Use in Phase 4, but remove or avoid leverage assumptions. |
| Robust profitability: execution quality/slippage | MERGE INTO NEW ROADMAP | Supports spread/slippage gates. | Use for Phase 3-4. |
| Robust profitability: post-trade review | MERGE INTO NEW ROADMAP | Matches journal/review phase. | Use for Phase 7/10 templates. |
| World Monitor awareness plan | KEEP | Correctly says external output is `research_trigger`, not trade signal. | Preserve as ingestion boundary. |
| EJLayer L16 risk policy | MERGE INTO NEW ROADMAP | Versioned risk policy exists and is useful. | Align with new 1%/no-leverage defaults. |
| EJLayer L22 realized slippage model | MERGE INTO NEW ROADMAP | Needed by roadmap risk/slippage rules. | Defer until risk engine has slippage fields. |
| EJLayer L24 broker latency tracking | PARK FOR LATER | Broker safety, but late-stage. | Defer to broker reconciliation phase. |
| Old live/IB quickstart and execution docs | PARK FOR LATER | Potentially useful operationally, risky now. | Keep archived; do not make active until tiny-live gate. |
| Generated/completed plan packs duplicated under `Docs/plans/Completed` | DELETE CANDIDATE | Duplicative historical packs clutter planning truth. | Do not delete yet; first add non-canonical index/notice. |

## C. New Implementation Sequence

1. Safety baseline
   - Make the new roadmap canonical.
   - Document current broker-capable paths.
   - Assert live and leverage are disabled.
   - Add tests around `ALLOW_LIVE_TRADING`, `IB_PAPER_TRADING`, `JAX_RUNTIME_MODE`, and `global_kill_switch`.

2. Structured trade candidate model
   - Extend candidate/domain contract to include catalyst, setup type, evidence summary, invalidation, slippage allowance, max loss fields, and status mapping.

3. Evidence scoring
   - Add structured evidence items with source, timestamp, confidence, impact, contradiction flag, and data quality score.

4. Trust gates / strict gatekeeper
   - Centralize hard rejects: no catalyst, no stop, no invalidation, stale data, poor spread/slippage, no sizing, leverage request, missing human approval.

5. Risk engine with slippage buffer
   - Decimal-safe sizing using `stop_distance + slippage_allowance`.
   - Calculate normal and slippage-adjusted max loss.

6. Paper approval loop
   - Keep current approval flow, but only after candidate completeness and gatekeeper pass.

7. Journal and review templates
   - Map the new review template to existing review/lesson/trade review storage.

8. Shadow mode
   - Produce real-time candidates and compare outcomes without any broker submission.

9. Tiny live activation only after explicit approval
   - Requires separate approval, broker reconciliation, slippage evidence, kill-switch proof, and no leverage.

## D. First Implementation Task

Add a Safety Baseline test/report that inventories execution-enabling controls and fails if defaults permit live trading or leverage.

This should not add trading behavior. It should codify current safety assumptions:

- `ALLOW_LIVE_TRADING` default is not enabled.
- Execution instruction worker only runs in `JAX_RUNTIME_MODE=paper`, `IB_PAPER_TRADING=true`, `ALLOW_LIVE_TRADING=false`.
- ETF catalog rejects live mode and leveraged/inverse/volatility ETFs.
- `config/risk-constraints.json` must stay at `max_leverage <= 1.0` for the current safety baseline.

## E. Files Likely To Change

- Modify or add tests near `cmd/trader/execution_instruction_worker_test.go`.
- Modify or add tests near `internal/modules/instruments/policy_test.go`.
- Add a small safety config test, likely `libs/risk/policy_test.go` or a new focused test under `tests/golden` / `tests/safety`.
- Possibly update `config/risk-constraints.json` to set leverage disabled, but only after deciding whether config changes are allowed in the first task.
- Update `Docs/ROADMAP.md` or `Docs/plans/README.md` to point at the new controlling roadmap.

## F. Test Plan

- Unit test: execution worker disabled unless all paper-only env gates are exactly satisfied.
- Unit test: `ALLOW_LIVE_TRADING=true` disables paper worker.
- Unit test: ETF catalog rejects `live` mode for approved ETFs.
- Unit test: ETF catalog rejects `TQQQ`, `SQQQ`, `UVXY`, `VXX`.
- Config test: fail or warn on leverage allowance during restricted phases.
- Optional doc test: verify active roadmap pointer names `Docs/plans/jax-trading-roadmap-pack`.

## G. Risks / Warnings

- `internal/modules/execution.Service` can call `PlaceOrder`; do not touch or expand it until safety baseline is locked.
- `/api/v1/execute` permits live mode if `ALLOW_LIVE_TRADING=true`; this must remain treated as dangerous and late-stage only.
- `EXECUTION_ENABLED=true` plus IB bridge config can initialize broker-capable execution.
- `config/risk-constraints.json` must not move above `max_leverage: 1.0` during the current safety-baseline phase.
- Archived autonomous docs explicitly describe automatic execution. They should be labeled historical/non-canonical before future agents follow them.
- Existing candidate model is useful but incomplete versus the new roadmap. Do not build more automation on top of it until the missing safety fields are first-class.

## Verification

This report was created from repo inspection only. No trading features were implemented. No tests were run for this docs-only file creation.

## What's Left

- Decide whether the first task may update config/docs or should be tests-only.
- Add a canonical notice pointing from old active roadmap/docs to the new roadmap pack.
- Run the first safety-baseline test task before any trading feature implementation.
