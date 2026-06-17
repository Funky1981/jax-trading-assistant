# 09 — Phased Codex Tickets

## Ticket 01 — Technical Analysis Snapshot Model

```text
Add technical_analysis_snapshots storage, models, validation, and tests.

Acceptance:
- stores trend, levels, event reaction, volume/ATR, relative strength
- calculates technical score
- returns verdict and reasons
- missing candles fail safely
```

## Ticket 02 — Technical Analysis Engine

```text
Build deterministic TA service.

Acceptance:
- detects pre-event high/low break
- checks VWAP hold/reject if VWAP available
- checks volume/ATR expansion
- checks relative strength vs SPY
- flags too_extended and whipsaw
```

## Ticket 03 — Fundamental Analysis Snapshot Model

```text
Add fundamental_analysis_snapshots storage, models, validation, and tests.

Acceptance:
- stores event summary, expected market impact, themes, cross-market checks, confounders, missing evidence
- calculates fundamental score
- headline is never the only evidence
```

## Ticket 04 — Fundamental Analysis Engine

```text
Build FA service.

Acceptance:
- CPI/jobs/Fed events produce expected market impact
- checks actual vs expected
- checks ETF relevance
- checks other events/confounders
- missing evidence is explicit
```

## Ticket 05 — Event Playbook Library

```text
Add playbooks for hot CPI, cool CPI, strong jobs, weak jobs, hawkish Fed, dovish Fed, AI/semi news, bank stress, oil shock.

Acceptance:
- event maps to playbook
- playbook maps allowlisted ETFs only
- playbook defines required TA/FA checks
- unknown event blocks candidate
```

## Ticket 06 — Analyst Scoring Service

```text
Combine TA, FA, risk, and confidence scores.

Acceptance:
- score persisted
- thresholds applied
- hard vetoes override score
- candidate_allowed only when all gates pass
```

## Ticket 07 — Multi-Analyst Review

```text
Create review flow: Fundamental Analyst, Technical Analyst, Risk Manager, Trade Reviewer.

Acceptance:
- each role has output
- final decision links to role outputs
- LLM summary cannot override veto
```

## Ticket 08 — Analyst Memory

```text
Store every event decision as a case study.

Acceptance:
- case study created from decision
- paper outcome can be attached
- feedback can be stored
- similar case retrieval works
```

## Ticket 09 — Evidence Bundle Integration

```text
Add TA/FA sections to macro evidence bundle.

Acceptance:
- evidence includes TA snapshot
- evidence includes FA snapshot
- evidence includes score and vetoes
- walk-away reasons included
```

## Ticket 10 — Backtesting and UAT Fixtures

```text
Add deterministic TA/FA fixtures.

Acceptance:
- hot CPI bearish fixture passes
- Fed whipsaw rejects
- confounder fixture rejects
- missing data rejects
- no broker write occurs
```
