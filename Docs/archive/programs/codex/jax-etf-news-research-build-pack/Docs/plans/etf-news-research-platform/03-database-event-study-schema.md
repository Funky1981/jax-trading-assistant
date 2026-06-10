# 03 — Database Event-Study Schema

## Goal

Extend existing Postgres schema to support historical ETF news research.

Do not create a second database.

## Existing Tables To Reuse

- `event_sources`
- `event_raw`
- `event_normalized`
- `event_symbol_map`
- `quotes`
- `candles`
- `candidate_trades`
- `candidate_events`
- `memory_items`

## New Tables

### 1. `event_windows`

Stores ETF price movement around an event.

```sql
CREATE TABLE event_windows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL,
    symbol TEXT NOT NULL,
    window_name TEXT NOT NULL,
    window_start TIMESTAMPTZ NOT NULL,
    window_end TIMESTAMPTZ NOT NULL,
    price_before NUMERIC,
    price_after NUMERIC,
    return_pct NUMERIC,
    benchmark_symbol TEXT,
    benchmark_return_pct NUMERIC,
    abnormal_return_pct NUMERIC,
    volume_before NUMERIC,
    volume_after NUMERIC,
    volume_change_pct NUMERIC,
    spread_before_bps NUMERIC,
    spread_after_bps NUMERIC,
    volatility_adjusted_move NUMERIC,
    data_quality TEXT NOT NULL DEFAULT 'unknown',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(event_id, symbol, window_name)
);
```

Indexes:

```sql
CREATE INDEX idx_event_windows_symbol_window ON event_windows(symbol, window_name);
CREATE INDEX idx_event_windows_event_symbol ON event_windows(event_id, symbol);
```

### 2. `event_confounders`

Stores other events that may have affected the same ETF movement.

```sql
CREATE TABLE event_confounders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL,
    confounding_event_id UUID NOT NULL,
    symbol TEXT NOT NULL,
    relationship_type TEXT NOT NULL,
    time_distance_minutes INT NOT NULL,
    relevance_score NUMERIC NOT NULL,
    reason TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(event_id, confounding_event_id, symbol)
);
```

Relationship types:

```text
macro
sector
company
geopolitical
earnings
rates
commodity
credit
unknown
```

### 3. `event_priced_in_scores`

Stores priced-in verdicts.

```sql
CREATE TABLE event_priced_in_scores (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL,
    symbol TEXT NOT NULL,
    pre_event_1h_return NUMERIC,
    pre_event_4h_return NUMERIC,
    pre_event_1d_return NUMERIC,
    post_event_15m_return NUMERIC,
    post_event_1h_return NUMERIC,
    benchmark_symbol TEXT,
    benchmark_return NUMERIC,
    abnormal_return NUMERIC,
    volume_confirmation_score NUMERIC,
    spread_quality_score NUMERIC,
    priced_in_score NUMERIC NOT NULL,
    verdict TEXT NOT NULL,
    reason TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(event_id, symbol)
);
```

Verdicts:

```text
not_priced_in
partially_priced_in
priced_in
overreaction
unclear
```

### 4. `etf_context_snapshots`

Stores ETF research context.

```sql
CREATE TABLE etf_context_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    symbol TEXT NOT NULL,
    theme TEXT NOT NULL,
    sector TEXT,
    benchmark_symbol TEXT,
    related_symbols JSONB NOT NULL DEFAULT '[]',
    macro_sensitivity JSONB NOT NULL DEFAULT '{}',
    notes TEXT,
    valid_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_to TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### 5. `research_summaries`

Stores precomputed approval-ready summaries.

```sql
CREATE TABLE research_summaries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL,
    symbol TEXT NOT NULL,
    strategy_id TEXT,
    summary TEXT NOT NULL,
    why_this_etf TEXT NOT NULL,
    what_happened TEXT NOT NULL,
    what_else_mattered TEXT,
    priced_in_view TEXT NOT NULL,
    risk_notes TEXT NOT NULL,
    evidence JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

## Migration Rules

- Create additive migrations only.
- Use idempotent indexes where safe.
- Add constraints for verdict/category enums where practical.
- Add down migration.
- Add integration tests that verify tables, indexes, and unique constraints.

## Acceptance Criteria

- Migrations apply cleanly from empty DB.
- Migrations apply cleanly to existing DB.
- Event window rows are unique per event/symbol/window.
- Priced-in score is unique per event/symbol.
- Confounders are queryable by event and symbol.
