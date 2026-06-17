# 08 — Walk-Away Engine

## Goal

Make Jax excellent at saying no.

Most events should not become trades.

## Walk-away categories

```text
fundamental_unclear
technical_no_confirmation
cross_asset_conflict
regime_conflict
major_confounder
priced_in
move_too_extended
no_clean_stop
bad_reward_risk
poor_liquidity
risk_limit_hit
strategy_not_matched
missing_data
operator_caution
```

## Data model

### walkaway_decisions

```sql
CREATE TABLE walkaway_decisions (
    id UUID PRIMARY KEY,
    event_id UUID NULL,
    symbol TEXT NOT NULL,
    category TEXT NOT NULL,
    severity TEXT NOT NULL,
    reason TEXT NOT NULL,
    evidence_refs JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

## Severity

```text
info
warning
blocker
critical
```

## Behaviour

```text
blocker = no candidate
critical = no candidate and alert operator
warning = reduce score/risk
info = evidence only
```

## Example output

```text
No trade: QQQ sold off after CPI, but TLT rallied and yields proxy did not confirm. Cross-asset conflict means the CPI thesis is not clean.
```

## Codex task

```text
Create Walk-Away Engine.

Inputs:
- all analysis snapshots
- hard vetoes
- missing evidence
- failed strategy rules

Outputs:
- walk-away decisions
- human-readable no-trade summary
```

## Tests

```text
cross-asset conflict creates blocker
move too extended creates blocker/watch-only
missing non-critical evidence creates warning
critical broker/data issue creates critical
multiple walk-away reasons aggregate cleanly
```

## Acceptance criteria

```text
no-trade decisions are stored
UI can display why Jax walked away
blockers prevent candidate creation
