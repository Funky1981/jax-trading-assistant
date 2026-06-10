# 10 — Phased Codex Tickets

## Ticket 01 — Add macro event storage

```text
Create macro_events and macro_event_etf_map migrations, models, validation, and tests.

Acceptance:
- valid NFP/CPI/Fed events persist
- invalid events reject/quarantine
- ETF map enforces allowlist
- no candidate/order/trade created
```

## Ticket 02 — Add macro event service

```text
Create service that accepts validated macro research triggers and normalises them into macro_events.

Acceptance:
- source_event_id dedupe
- surprise_value and surprise_percent calculated
- direction classified
- mapped ETFs persisted
```

## Ticket 03 — Add chart reaction snapshots

```text
Create service that fetches pre/post candles for mapped ETFs and persists reaction snapshots.

Acceptance:
- supports 5m/15m/30m/60m windows
- confirms expected direction
- flags too_extended
- flags noisy/whipsaw
- missing candles block candidate
```

## Ticket 04 — Add scenario playbooks

```text
Create deterministic scenario playbooks for hot CPI, cool CPI, strong jobs, weak jobs, hawkish Fed, dovish Fed.

Acceptance:
- each playbook maps ETFs
- each playbook defines expected reaction
- each playbook defines required confirmation windows
- unknown scenarios block candidate
```

## Ticket 05 — Add priced-in and confounder checks

```text
Create simple deterministic priced-in scoring and confounder detection.

Acceptance:
- priced_in blocks candidate
- unclear blocks candidate
- high-severity confounder blocks candidate
- reasons are stored
```

## Ticket 06 — Add evidence bundle builder

```text
Create macro evidence bundles.

Acceptance:
- bundle includes macro facts, chart reaction, scenario, priced-in verdict, confounders, risk, walk-away reasons
- candidate_allowed only when all gates pass
- missing evidence is explicit
```

## Ticket 07 — Add candidate trade generator

```text
Create paper-only candidate from candidate_allowed evidence bundle.

Acceptance:
- no candidate from blocked/watch/insufficient bundles
- no broker order
- risk cap enforced
- status awaiting_human_approval
```

## Ticket 08 — Add API endpoints

```text
Expose macro events, reactions, evidence, candidates.

Acceptance:
- protected API routes
- list/detail endpoints
- candidate approval/rejection routes reuse existing approval guard
- no direct execution endpoint added
```

## Ticket 09 — Add frontend screens

```text
Add Macro Event Inbox, Event Detail, Candidate Review.

Acceptance:
- shows actual vs expected
- shows mapped ETFs
- shows chart reaction summary
- shows evidence and no-trade reasons
- manual approval boundary is obvious
```

## Ticket 10 — Add backtest/UAT fixtures

```text
Add deterministic event/candle fixtures and UAT checklist.

Acceptance:
- hot CPI bearish QQQ fixture passes
- cool CPI bullish QQQ fixture passes
- whipsaw Fed fixture rejects
- priced-in fixture rejects
- no broker write test passes
```
