# 01 — Target Architecture

## Product Goal

Jax should become an ETF-only, news-driven research and paper-trading assistant.

## End-to-End Flow

```text
news / macro event detected
    ↓
event normalized and stored
    ↓
ETF relevance mapping
    ↓
historical event-study lookup
    ↓
priced-in scoring
    ↓
confounder analysis
    ↓
evidence bundle generated
    ↓
strategy evaluates candidate
    ↓
guardrails check
    ↓
AI summarises evidence
    ↓
mobile/frontend approval
    ↓
paper order instruction
    ↓
order/position tracking
    ↓
exit by stop/take-profit/flatten
    ↓
memory retain + reflection
```

## Major Components

### 1. Event Ingestion

Responsible for:

- historical news backfill
- live news later
- macro calendar events
- ETF/sector/constituent mapping

### 2. Event Study Engine

Responsible for:

- price windows before and after news
- return calculations
- volume change
- spread change
- abnormal return vs benchmark
- volatility-adjusted movement

### 3. Confounder Engine

Responsible for:

- finding overlapping news
- ranking possible alternate causes
- identifying macro/sector/company overlap
- storing explainable relevance reasons

### 4. Priced-In Engine

Responsible for:

- detecting pre-event drift
- comparing pre/post movement
- classifying whether the move is already priced in
- blocking poor risk/reward candidates

### 5. Evidence Bundle Builder

Responsible for producing a structured research object containing:

- news summary
- selected ETF
- why this ETF
- historical similar cases
- price reaction
- other news/confounders
- priced-in verdict
- risk notes
- walk-away checks

### 6. AI Research Layer

Responsible for:

- summarising evidence
- ranking candidate quality
- explaining the trade in beginner language
- recommending trade/wait/reject

Not allowed to:

- submit orders directly
- override hard guardrails
- invent missing data
- approve trades

### 7. Approval and Paper Execution

Responsible for:

- sending mobile alert
- receiving approval/rejection
- expiring stale candidates
- creating paper-only execution instruction
- recording audit trail

## Source of Truth Rules

- Postgres is source of truth.
- Provider APIs are external inputs, not truth.
- AI output is advisory, not truth.
- Mobile approvals are decisions, not trades.
- Broker fills are execution truth.
- Memory is supporting context, not primary data.

## Architecture Rule

All strategy decisions must be reproducible from stored data.
