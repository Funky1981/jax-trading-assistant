# 12 — Codex Implementation Prompts

Use these prompts in order. Do not skip the blocker prompts.

## Prompt 0 — Fix Current Branch Blockers First

```text
Review the current Jax `work` branch and fix blockers before adding swing mode.

Scope:
- Fix `gofmt` issue reported for `internal/modules/tradingmodule/module.go`.
- Fix event-study bounds so historical studies use selected event times, not `time.Now()`.
- Remove duplicate ETF allowlist map from `cmd/research/backfill.go` and centralise phase-one ETF policy.
- Replace hardcoded `StaleQuotePass: true` and `PaperModePass: true` in research evidence bundles with explicit runtime guardrail evidence or `not_evaluated` state for historical research.
- Wire confounder detection so event-study can persist and include confounders.

Constraints:
- Do not enable live trading.
- Do not change broker execution behaviour except to make paper/live proof explicit.
- Preserve cmd/trader vs cmd/research boundary.

Tests:
- Add unit tests for event-study bounds using historical event times.
- Add tests proving non-allowlisted ETF rejects through central policy.
- Add tests proving missing runtime quote cannot become stale_quote_pass=true.
- Add tests proving confounders appear in evidence bundle.

Run:
.\scripts\go-verify.ps1 -Mode quick
```

## Prompt 1 — Add Trading Horizons and Mode Catalog

```text
Add explicit trading horizons and modes for Jax.

Create/extend:
- internal/modules/tradingmodes
- config/trading-modes/swing-research.json
- config/trading-modes/swing-paper.json
- config/trading-modes/intraday-paper.json

Modes:
- swing_research
- swing_paper
- intraday_paper

Add horizon policy contract:
- horizon
- holdPeriodTargetDays
- maxHoldDays
- flattenByClose
- overnightRiskAllowed
- weekendHoldAllowed
- requiresDailyReview
- revalidationSchedule
- thesisInvalidators

Acceptance:
- Intraday mode requires flattenByClose=true and overnightRiskAllowed=false.
- Swing mode requires overnightRiskAllowed=true and requiresDailyReview=true.
- All modes are ETF-only and paper-only in phase 1.
```

## Prompt 2 — Add Swing Data Model

```text
Add database migrations for:
- research_theses
- candidate_horizon_policies
- candidate_revalidation_checks
- guardrail_evaluations
- event_confounders if missing/incomplete
- ai_research_audit

Acceptance:
- Migrations apply to empty database.
- Migrations apply to current development database.
- Invalid horizon values reject.
- Swing candidate horizon policy can be linked to candidate.
- Revalidation checks can be recorded before and after paper execution.
```

## Prompt 3 — Extend Event Study With Swing Windows

```text
Extend cmd/research event study to include swing windows:
- event_to_+1d
- event_to_+2d
- event_to_+3d
- event_to_+5d
- event_to_+10d

Add swing edge score:
- sample count
- median returns
- win rates
- max adverse excursion
- follow-through quality
- reversal risk
- data quality

Acceptance:
- Existing intraday event-study tests still pass.
- New swing windows compute correctly.
- Missing data fails closed.
- Event bounds use historical event times.
```

## Prompt 4 — Add Swing Research Thesis Engine

```text
Build a swing thesis engine in cmd/research.

Input:
- normalized event
- affected ETF
- event-study results
- priced-in score
- confounders
- AI/provider enrichment if configured

Output:
- research_thesis row
- evidence bundle JSON
- candidate eligibility decision

Strategies:
- etf_swing_macro_rates_rotation_v1
- etf_swing_sector_event_momentum_v1
- etf_swing_risk_off_risk_on_reversal_v1

Acceptance:
- Thesis can be `watch`, `avoid`, or `candidate_eligible`.
- Candidate eligibility requires no hard reject.
- Swing thesis includes invalidators and daily revalidation schedule.
```

## Prompt 5 — Wire Candidate Creation for Swing Paper Mode

```text
Wire cmd/trader so eligible swing theses can become paper candidates.

Requirements:
- Candidate horizon policy required.
- Evidence bundle required.
- Guardrail evaluation required.
- Paper mode proof required before paper instruction.
- Approval required before paper instruction.
- No live execution path.

Acceptance:
- Swing paper candidate cannot execute without approval.
- Swing paper candidate cannot execute without paper mode proof.
- Swing paper candidate cannot execute without daily revalidation schedule.
```

## Prompt 6 — Preserve Optional Intraday Paper Mode

```text
Preserve intraday paper mode as secondary.

Requirements:
- Same-session expiry.
- Flatten by close.
- No overnight risk.
- RTH and fresh quote required.
- Existing ETF news intraday strategies continue to work as intraday only.

Acceptance:
- Intraday and swing candidates are visibly different in API payloads and UI.
- Intraday cannot accidentally hold overnight.
- Swing does not inherit flatten-by-close.
```

## Prompt 7 — Add AI Provider Abstraction

```text
Add provider-agnostic AI research interface.

Providers:
- mock_test_provider
- ollama
- deepseek
- openai compatible adapter

Use AI only for:
- summarisation
- classification support
- ETF mapping suggestions
- evidence explanation
- beginner summaries

Acceptance:
- Provider can be changed by config.
- Invalid JSON fails closed.
- AI output containing execution instruction is rejected.
- AI audit row is persisted for cloud calls.
```

## Prompt 8 — Add Frontend Swing Research UX

```text
Add/update frontend pages:
- research inbox
- swing theses
- swing thesis detail
- candidate evidence page
- revalidation page
- trading mode picker

Acceptance:
- Swing vs intraday labels are clear.
- Approval page shows overnight risk for swing candidates.
- User can reject with reason.
- Missing evidence blocks approval.
```

## Prompt 9 — Add Swing UAT Scripts and Evidence Report

```text
Add UAT and evidence scripts:
- scripts/uat-etf-swing-research.ps1
- scripts/etf-swing-evidence-report.ps1

Flow must prove:
- services healthy
- broker paper mode proof
- event ingestion
- event study
- priced-in score
- confounders
- swing thesis
- candidate
- approval
- paper execution instruction
- revalidation
- reflection

Acceptance:
- Evidence artifacts written under Docs/runs/etf-swing-research.
- Run fails if any required evidence is missing.
```

## Prompt 10 — Final Production Readiness Gate

```text
Update Docs/PRODUCTION_READINESS.md and Docs/UAT_PAPER_TRADING.md for swing-first Jax.

Release remains blocked unless:
- gofmt passes
- all tests pass
- services healthy
- paper mode proof fresh
- no live trading proof fresh
- post-trade reflection fresh
- swing revalidation proof fresh
- operator/engineering/risk signoff recorded

Do not mark production-ready if any blocker remains.
```
