# 01 — Target Architecture

## Target System

Jax should become:

```text
Event-driven ETF swing research system
with optional intraday paper-trade mode
```

The architecture must keep the existing separation:

```text
cmd/research
  - ingest events
  - classify events
  - run event studies
  - compute priced-in scores
  - detect confounders
  - build research evidence
  - compare intraday vs swing windows
  - write research summaries and thesis records

cmd/trader
  - expose frontend APIs
  - manage candidate lifecycle
  - enforce ETF/risk/broker/paper guardrails
  - handle approval workflow
  - create paper execution instructions only after approval
  - record trade lifecycle and reflections

services/ib-bridge
  - broker and market connectivity
  - explicit paper/live mode proof

services/agent0-service
  - assistant/planning support
  - no direct trading authority

frontend
  - mode selection
  - research inbox
  - evidence review
  - approval/rejection
  - post-trade review
```

## Core Rule

Research can be broad. Trading must be narrow.

```text
AI/research output -> evidence bundle -> candidate -> guardrails -> human approval -> paper execution
```

There must be no path:

```text
AI/research output -> broker order
```

## Runtime Boundaries

### Research Runtime Responsibilities

`cmd/research` owns:

- World Monitor/Jax trigger ingestion endpoint or internal queue consumer.
- Historical event/candle backfill.
- Event classification and ETF mapping.
- Historical event-study windows.
- Priced-in scoring.
- Confounder detection.
- Swing thesis generation.
- Intraday alternative research.
- Memory/reflection write proposals.

It must not:

- Submit broker orders.
- Create execution instructions.
- Bypass candidate approval.
- Assume runtime guardrails passed.

### Trader Runtime Responsibilities

`cmd/trader` owns:

- Candidate creation and qualification.
- Approval state machine.
- Paper execution instruction creation.
- Paper broker submission orchestration.
- Runtime guardrails.
- Position/trade lifecycle.
- Close/cancel/protect actions.
- UAT readiness APIs.

It must not:

- Import research-only dependencies.
- Depend on external AI to enforce risk.
- Accept candidate payloads without validating mode/policy/ETF/risk.

## Primary Data Flow

```text
1. Event arrives
   source: World Monitor, news provider, calendar, manual import

2. Research runtime validates and stores raw event
   event_raw
   event_normalized
   event_source_quality

3. Research runtime classifies
   event_type
   affected_etfs
   source_count
   source_quality
   time_sensitivity

4. Research runtime studies history
   intraday windows: +5m, +15m, +1h, +4h, close
   swing windows: +1d, +2d, +3d, +5d, +10d

5. Research runtime detects confounders
   same-symbol events
   macro events
   market-wide moves
   related ETF/sector events

6. Research runtime builds thesis
   swing thesis first
   optional intraday setup second

7. Trader runtime creates candidate only if eligible
   evidence bundle required
   guardrails required
   paper-only required

8. Human approves/rejects

9. Paper execution instruction created only after approval

10. Daily revalidation runs for swing positions

11. Close/reflection/outcome recorded
```

## New Domain Concepts

Add these explicit concepts:

```text
TradingMode
TradingHorizon
ResearchTrigger
ResearchThesis
EvidenceBundle
CandidateTrade
GuardrailEvaluation
RevalidationCheck
ReflectionRecord
```

## Modes

```text
research_only
  no candidates, no execution

intraday_paper
  same-session paper candidate
  flatten by close required

swing_research
  multi-day thesis, no execution

swing_paper
  multi-day paper candidate
  overnight risk allowed
  daily revalidation required
```

## Production Wiring Requirements

Every path must be testable:

- API handler test.
- Domain unit test.
- Persistence integration test.
- UAT script proof.
- Evidence artifact output.

Every important decision must be auditable:

- source event id
- model/provider used
- classification reason
- affected ETF mapping reason
- event-study window results
- priced-in verdict
- confounders
- candidate decision
- human approval/rejection
- execution instruction id
- post-trade reflection id
