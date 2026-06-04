# 11 — Token-Saving Playbook

## Purpose

Reduce paid API usage in Jax without weakening trading safety, evidence quality, or beginner explainability.

## Golden Rule

```text
Do not send data to the model unless a deterministic gate proves the model is needed.
```

## Token-Saving Hierarchy

```text
1. Do not call AI.
2. If AI is needed, send less context.
3. If context is needed, compact it deterministically.
4. If bulky context remains, compress only safe sections.
5. If a model is needed, use the cheapest safe model.
6. If repeated, cache it.
7. If uncertain, escalate rarely.
```

## Jax-Specific Rule

```text
Compress supporting context.
Never compress trading truth.
```

## Trading Truth Must Stay Uncompressed

```text
symbol
asset_type
event_id
candidate_id
strategy_id
event timestamp
source timestamp
provider timestamp
current quote timestamp
entry price
stop-loss
take-profit
risk amount
position size
spread
quote freshness
priced-in verdict
guardrail pass/fail state
approval state
approval expiry
broker order id
fill state
paper/live mode
```

## Compressible Context

```text
news article body
duplicated headlines
search results
long source excerpts
historical research notes
backfill logs
test output
build logs
RAG chunks
similar event explanations
post-trade narrative notes
developer docs
```

## Main Savings Patterns

### 1. AI Eligibility Gate

Before every model call, Jax must check:

```text
ETF-relevant?
ETF allowlisted?
event recent?
event duplicate?
market/session valid?
quote fresh?
spread acceptable?
priced-in verdict acceptable?
guardrails passing?
```

If any hard gate fails:

```text
NO AI CALL
```

### 2. Event Clustering

Group duplicate/provider-overlapping news into one canonical event.

### 3. Evidence Bundles Only

Never send raw article/candle/log dumps to paid models.

### 4. Summary Ladders

```text
100 news items
  ↓
10 event clusters
  ↓
3 ETF-relevant themes
  ↓
1 evidence bundle
  ↓
1 approval summary
```

### 5. Strict Retrieval Limits

Default limits:

```text
similar_events_limit = 3
memory_limit = 5
confounder_limit = 3
source_excerpt_limit = 2
max_context_tokens = 4000
```

### 6. Template-Based Outputs

Use templates so models only fill small fields.

### 7. Deterministic No-Trade Summaries

Many no-trade explanations do not need AI.

### 8. Offline Batch AI

Use paid/strong models after market close for daily digest, reflection, calibration, missed opportunities, and historical clustering.

## Acceptance Criteria

- Most rejected events use zero paid AI calls.
- Every model call has a task type.
- Every model call has a cost record.
- Every model call has a reason for being eligible.
- Approval summaries use evidence bundles only.
- Trading truth is never compressed.
