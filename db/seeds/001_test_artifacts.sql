-- Seed: test artifacts for local development and integration tests
-- ADR-0012 Phase 6: provides a known-good APPROVED artifact for smoke testing
--
-- Schema source: db/postgres/006_strategy_artifacts.sql
-- Tables used:   strategy_artifacts, artifact_approvals
--
-- NOTE: artifact_approvals.artifact_id is a UUID FK to strategy_artifacts(id),
--       so the approval insert uses a subquery to resolve the UUID from the
--       human-readable artifact_id string.

-- ============================================================================
-- Strategy artifact
-- ============================================================================
INSERT INTO strategy_artifacts (
    artifact_id,
    schema_version,
    strategy_name,
    strategy_version,
    params,
    risk_profile,
    hash,
    payload,
    created_by,
    created_at
)
VALUES (
    'strat_rsi_momentum_2026-02-01T00:00:00Z',
    '1.0.0',
    'rsi_momentum',
    '1.0.0',
    '{"rsi_period": 14, "overbought": 70, "oversold": 30, "entry_threshold": 0.6}'::jsonb,
    '{"max_position_pct": 0.05, "max_daily_loss": 0.02, "allowed_order_types": ["MKT", "LMT"]}'::jsonb,
    'a3f8c2d1e4b7a9f0c2e5d8b1a4f7c0e3d6b9a2f5c8e1d4b7a0f3c6e9d2b5a8f1',
    '{
        "artifact_id": "strat_rsi_momentum_2026-02-01T00:00:00Z",
        "schema_version": "1.0.0",
        "strategy_name": "rsi_momentum",
        "strategy_version": "1.0.0",
        "params": {"rsi_period": 14, "overbought": 70, "oversold": 30, "entry_threshold": 0.6},
        "risk_profile": {"max_position_pct": 0.05, "max_daily_loss": 0.02, "allowed_order_types": ["MKT", "LMT"]},
        "created_by": "seed",
        "created_at": "2026-02-01T00:00:00Z"
    }'::jsonb,
    'seed',
    '2026-02-01 00:00:00+00'
)
ON CONFLICT (artifact_id) DO NOTHING;

-- ============================================================================
-- Approval record
-- artifact_approvals.artifact_id is UUID FK → strategy_artifacts.id
-- ============================================================================
INSERT INTO artifact_approvals (
    artifact_id,
    state,
    approved_by,
    approved_at,
    review_notes,
    reviewer,
    reviewed_at,
    validation_passed,
    state_changed_by,
    state_changed_at
)
SELECT
    sa.id,
    'APPROVED'::artifact_approval_state,
    'seed',
    '2026-02-01 00:00:00+00',
    'Seeded for local development and integration testing',
    'seed',
    '2026-02-01 00:00:00+00',
    TRUE,
    'seed',
    '2026-02-01 00:00:00+00'
FROM strategy_artifacts sa
WHERE sa.artifact_id = 'strat_rsi_momentum_2026-02-01T00:00:00Z'
ON CONFLICT (artifact_id) DO NOTHING;
