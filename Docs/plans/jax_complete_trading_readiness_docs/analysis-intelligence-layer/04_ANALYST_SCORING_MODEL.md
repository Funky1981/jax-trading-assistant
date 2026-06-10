# 04 — Analyst Scoring Model

## Goal

Combine technical, fundamental, and risk evidence into one disciplined decision.

Jax should not trade because one area looks good.

A candidate requires:

```text
fundamental pass
technical pass
risk pass
```

## Score components

```text
fundamental_score: 0–100
technical_score:   0–100
risk_score:        0–100
confidence_score:  0–100
```

## Combined score

Suggested formula:

```text
candidate_score =
  fundamental_score * 0.40 +
  technical_score   * 0.40 +
  risk_score        * 0.15 +
  confidence_score  * 0.05
```

## Thresholds

```text
0–49   reject
50–64  watch only
65–74  weak candidate / needs manual caution
75–84  candidate allowed
85+    strong candidate allowed
```

## Hard vetoes

Any of these means no candidate:

```text
no chart confirmation
fundamental verdict conflicted
priced-in verdict priced_in
priced-in verdict unclear
major confounder unresolved
no stop level
reward:risk below minimum
ETF not allowlisted
market data missing
source quality too low
live trading requested
```

## Technical score breakdown

```text
Trend alignment:             0–20
Level break/hold quality:    0–20
Event reaction quality:      0–20
Volume/ATR confirmation:     0–15
Relative strength:           0–15
Entry/stop quality:          0–10
```

## Fundamental score breakdown

```text
Event surprise size:          0–20
Policy/rates impact:          0–20
ETF relevance:                0–15
Expectation/priced-in clarity:0–15
Cross-market confirmation:    0–15
Confounder cleanliness:       0–10
Source quality:               0–5
```

## Risk score breakdown

```text
Defined stop:                 0–25
Reward:risk:                  0–25
Position size safety:         0–20
Chase/extension control:      0–15
Correlation/exposure control: 0–15
```

## Confidence score

```text
data completeness
source agreement
historical support
model consistency
operator feedback history
```

## Decision values

```text
candidate_allowed
candidate_rejected
watch_only
insufficient_evidence
manual_review_only
```

## Data model

### analyst_decisions

```sql
CREATE TABLE analyst_decisions (
    id UUID PRIMARY KEY,
    macro_event_id UUID NULL,
    symbol TEXT NOT NULL,
    fundamental_snapshot_id UUID NULL,
    technical_snapshot_id UUID NULL,
    evidence_bundle_id UUID NULL,
    fundamental_score NUMERIC NOT NULL,
    technical_score NUMERIC NOT NULL,
    risk_score NUMERIC NOT NULL,
    confidence_score NUMERIC NOT NULL,
    candidate_score NUMERIC NOT NULL,
    decision TEXT NOT NULL,
    hard_vetoes TEXT[] NOT NULL DEFAULT '{}',
    reasons TEXT[] NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

## Codex task

```text
Create analyst scoring service.

Inputs:
- technical snapshot
- fundamental snapshot
- risk result
- priced-in score
- confounders

Outputs:
- analyst decision
- candidate score
- hard vetoes
- reasons
```

## Tests

```text
high FA + high TA + good risk = candidate_allowed
high FA + low TA = watch_only or rejected
high TA + conflicted FA = rejected
priced_in veto overrides high score
no stop veto overrides high score
missing data creates insufficient_evidence
```

## Acceptance criteria

```text
candidate score persisted
hard vetoes enforced
decision explainable
candidate generator can only use candidate_allowed
```
