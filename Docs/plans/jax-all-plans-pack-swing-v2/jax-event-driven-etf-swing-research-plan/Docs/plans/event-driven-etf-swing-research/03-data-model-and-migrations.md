# 03 — Data Model and Migrations

## Goal

Add swing-first research and revalidation without corrupting existing candidate/approval flow.

## Migration 1 — Research Thesis Table

Create:

```sql
CREATE TABLE IF NOT EXISTS research_theses (
    id UUID PRIMARY KEY,
    source_event_id UUID NOT NULL,
    symbol TEXT NOT NULL,
    strategy_id TEXT NOT NULL,
    horizon TEXT NOT NULL CHECK (horizon IN ('intraday','swing')),
    thesis_status TEXT NOT NULL,
    thesis_direction TEXT NOT NULL CHECK (thesis_direction IN ('long','short','avoid','watch')),
    confidence NUMERIC(6,4) NOT NULL,
    event_type TEXT NOT NULL,
    headline TEXT NOT NULL,
    summary TEXT,
    why_this_etf TEXT NOT NULL,
    historical_edge_summary JSONB NOT NULL DEFAULT '{}',
    priced_in_summary JSONB NOT NULL DEFAULT '{}',
    confounder_summary JSONB NOT NULL DEFAULT '{}',
    risk_summary JSONB NOT NULL DEFAULT '{}',
    thesis_invalidators JSONB NOT NULL DEFAULT '[]',
    hold_period_target_days INTEGER,
    max_hold_days INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_research_theses_event_symbol ON research_theses(source_event_id, symbol);
CREATE INDEX IF NOT EXISTS idx_research_theses_status ON research_theses(thesis_status);
CREATE INDEX IF NOT EXISTS idx_research_theses_horizon ON research_theses(horizon);
```

## Migration 2 — Candidate Horizon Policy

Extend candidate table, or add side table if candidate schema must remain stable:

```sql
CREATE TABLE IF NOT EXISTS candidate_horizon_policies (
    candidate_id UUID PRIMARY KEY,
    horizon TEXT NOT NULL CHECK (horizon IN ('intraday','swing')),
    hold_period_target_days INTEGER,
    max_hold_days INTEGER,
    flatten_by_close BOOLEAN NOT NULL,
    overnight_risk_allowed BOOLEAN NOT NULL,
    weekend_hold_allowed BOOLEAN NOT NULL DEFAULT false,
    requires_daily_review BOOLEAN NOT NULL DEFAULT false,
    revalidation_schedule TEXT,
    thesis_invalidators JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

## Migration 3 — Revalidation Checks

Create:

```sql
CREATE TABLE IF NOT EXISTS candidate_revalidation_checks (
    id UUID PRIMARY KEY,
    candidate_id UUID NOT NULL,
    trade_id UUID,
    check_type TEXT NOT NULL,
    check_status TEXT NOT NULL CHECK (check_status IN ('pass','warn','fail','not_applicable')),
    checked_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    summary TEXT NOT NULL,
    evidence JSONB NOT NULL DEFAULT '{}',
    action_required BOOLEAN NOT NULL DEFAULT false,
    recommended_action TEXT,
    created_by TEXT NOT NULL DEFAULT 'jax'
);

CREATE INDEX IF NOT EXISTS idx_candidate_revalidation_candidate ON candidate_revalidation_checks(candidate_id, checked_at DESC);
```

## Migration 4 — Runtime Guardrail Evidence

Create:

```sql
CREATE TABLE IF NOT EXISTS guardrail_evaluations (
    id UUID PRIMARY KEY,
    candidate_id UUID,
    thesis_id UUID,
    evaluation_scope TEXT NOT NULL,
    passed BOOLEAN NOT NULL,
    hard_reject BOOLEAN NOT NULL DEFAULT false,
    failures JSONB NOT NULL DEFAULT '[]',
    checks JSONB NOT NULL DEFAULT '{}',
    quote_timestamp TIMESTAMPTZ,
    market_session TEXT,
    broker_mode TEXT,
    evaluated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_guardrail_evaluations_candidate ON guardrail_evaluations(candidate_id, evaluated_at DESC);
```

## Migration 5 — Confounder Records

If not already present in usable shape, create:

```sql
CREATE TABLE IF NOT EXISTS event_confounders (
    id UUID PRIMARY KEY,
    event_id UUID NOT NULL,
    confounding_event_id UUID NOT NULL,
    symbol TEXT NOT NULL,
    relationship_type TEXT NOT NULL,
    relevance_score NUMERIC(6,4) NOT NULL,
    notes TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(event_id, confounding_event_id, symbol, relationship_type)
);
```

## Migration 6 — Model/Provider Audit

Create:

```sql
CREATE TABLE IF NOT EXISTS ai_research_audit (
    id UUID PRIMARY KEY,
    scope TEXT NOT NULL,
    entity_id UUID NOT NULL,
    provider TEXT NOT NULL,
    model TEXT NOT NULL,
    prompt_version TEXT NOT NULL,
    input_hash TEXT NOT NULL,
    output_hash TEXT NOT NULL,
    token_input INTEGER,
    token_output INTEGER,
    cost_estimate_usd NUMERIC(12,6),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

## Data Integrity Rules

- Candidate cannot exist without an evidence bundle id.
- Swing candidate cannot exist without a horizon policy.
- Swing execution cannot proceed unless a revalidation schedule exists.
- Intraday execution cannot proceed if `flatten_by_close=false`.
- Paper instruction cannot be created unless guardrail evaluation passed.
- Live broker mode hard-rejects in phase 1.

## Rollback Rules

Each migration must have safe rollback instructions in comments or a paired down migration if the repo supports it.

## Tests

- Migration applies cleanly to empty DB.
- Migration applies cleanly to existing branch DB.
- Unique constraints prevent duplicate confounders.
- Candidate with invalid horizon rejects.
- Revalidation rows can link to candidate before trade id exists.
