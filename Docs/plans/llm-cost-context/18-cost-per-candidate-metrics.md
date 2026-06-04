# 17 — Cost Per Candidate Metrics

## Purpose

Measure whether Jax is economically sane.

Total token count is useful, but the better metric is:

```text
cost per useful candidate
```

## Track

```text
cost_per_news_event
cost_per_rejected_event
cost_per_candidate
cost_per_approved_paper_trade
cost_per_strategy
cost_per_etf
cost_per_profitable_paper_trade
cost_per_false_positive
```

## Suggested Table

```sql
CREATE TABLE llm_cost_rollups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rollup_type TEXT NOT NULL,
    rollup_key TEXT NOT NULL,
    event_count INT NOT NULL DEFAULT 0,
    candidate_count INT NOT NULL DEFAULT 0,
    approved_count INT NOT NULL DEFAULT 0,
    total_input_tokens INT NOT NULL DEFAULT 0,
    total_output_tokens INT NOT NULL DEFAULT 0,
    total_cost_usd NUMERIC NOT NULL DEFAULT 0,
    from_ts TIMESTAMPTZ NOT NULL,
    to_ts TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

## Dashboard Metrics

Show:

```text
today spend
month spend
cost per candidate
cost per approved trade
paid calls avoided
cache hit rate
Headroom token savings
events rejected before AI
top expensive workflows
```

## Go / No-Go

A strategy should be disabled or reviewed if:

```text
high API cost
low candidate quality
poor paper outcome
high false positives
high strong-model escalation rate
```

## Acceptance Criteria

- Cost per candidate visible.
- Cost per strategy visible.
- Strong-model usage visible.
- Expensive workflows are identifiable.
