-- Seed: approved strategy artifact for local paper-trading and smoke tests
-- Keeps trader signal generation available in environments with an empty DB.

-- Remove the stale revoked e2e fixture so the bootstrap artifact is the latest
DELETE FROM artifact_approvals
WHERE artifact_id IN (
    SELECT id
    FROM strategy_artifacts
    WHERE artifact_id = 'strat_rsi_momentum_e2e_2026-02-18T00:00:00Z'
);

DELETE FROM strategy_artifacts
WHERE artifact_id = 'strat_rsi_momentum_e2e_2026-02-18T00:00:00Z';

INSERT INTO strategy_artifacts (
    id,
    artifact_id,
    schema_version,
    strategy_name,
    strategy_version,
    code_ref,
    params,
    data_window_from,
    data_window_to,
    symbols,
    validation,
    risk_profile,
    hash,
    payload,
    created_by,
    created_at,
    data_source_type,
    source_provider,
    dataset_id,
    dataset_hash,
    is_synthetic,
    synthetic_reason,
    provenance_verified_at
)
VALUES (
    '33333333-3333-3333-3333-333333333333',
    'strat_rsi_momentum_2026-03-11T10:20:00Z',
    '1.0.0',
    'rsi_momentum',
    '1.0.0',
    '',
    '{"entry_threshold": 0.6, "overbought": 70, "oversold": 30, "rsi_period": 14}'::jsonb,
    '2025-01-01 00:00:00+00',
    '2026-03-01 00:00:00+00',
    ARRAY['AAPL','MSFT','SPY'],
    '{
      "backtest_run_id": "22222222-2222-2222-2222-222222222222",
      "determinism_seed": 42,
      "metrics": {
        "max_drawdown": 0.08,
        "profit_factor": 1.9,
        "sharpe_ratio": 1.45,
        "total_return_pct": 18.2,
        "total_trades": 124,
        "win_rate": 0.58
      },
      "report_uri": "/reports/artifacts/rsi-momentum-2026-03-11.md"
    }'::jsonb,
    '{"allowed_order_types": ["MKT", "LMT"], "max_daily_loss": 0.02, "max_position_pct": 0.05}'::jsonb,
    'bc5d0541e88c0accd43ca1d33ccf74740ad0d163f83c77b21b38bdc6acc8dff1',
    '{
      "artifact_id": "strat_rsi_momentum_2026-03-11T10:20:00Z",
      "created_at": "2026-03-11T10:20:00Z",
      "created_by": "bootstrap",
      "data_window": {
        "from": "2025-01-01T00:00:00Z",
        "symbols": ["AAPL", "MSFT", "SPY"],
        "to": "2026-03-01T00:00:00Z"
      },
      "risk_profile": {
        "allowed_order_types": ["MKT", "LMT"],
        "max_daily_loss": 0.02,
        "max_position_pct": 0.05
      },
      "schema_version": "1.0.0",
      "strategy": {
        "code_ref": "",
        "name": "rsi_momentum",
        "params": {
          "entry_threshold": 0.6,
          "overbought": 70,
          "oversold": 30,
          "rsi_period": 14
        },
        "version": "1.0.0"
      },
      "validation": {
        "backtest_run_id": "22222222-2222-2222-2222-222222222222",
        "determinism_seed": 42,
        "metrics": {
          "max_drawdown": 0.08,
          "profit_factor": 1.9,
          "sharpe_ratio": 1.45,
          "total_return_pct": 18.2,
          "total_trades": 124,
          "win_rate": 0.58
        },
        "report_uri": "/reports/artifacts/rsi-momentum-2026-03-11.md"
      }
    }'::jsonb,
    'bootstrap',
    '2026-03-11 10:20:00+00',
    'real',
    'ib-bridge',
    'paper-bootstrap-rsi-momentum',
    'bootstrap-rsi-momentum-v1',
    FALSE,
    NULL,
    '2026-03-11 10:20:00+00'
)
ON CONFLICT (artifact_id) DO UPDATE SET
    schema_version = EXCLUDED.schema_version,
    strategy_name = EXCLUDED.strategy_name,
    strategy_version = EXCLUDED.strategy_version,
    code_ref = EXCLUDED.code_ref,
    params = EXCLUDED.params,
    data_window_from = EXCLUDED.data_window_from,
    data_window_to = EXCLUDED.data_window_to,
    symbols = EXCLUDED.symbols,
    validation = EXCLUDED.validation,
    risk_profile = EXCLUDED.risk_profile,
    hash = EXCLUDED.hash,
    payload = EXCLUDED.payload,
    created_by = EXCLUDED.created_by,
    created_at = EXCLUDED.created_at,
    data_source_type = EXCLUDED.data_source_type,
    source_provider = EXCLUDED.source_provider,
    dataset_id = EXCLUDED.dataset_id,
    dataset_hash = EXCLUDED.dataset_hash,
    is_synthetic = EXCLUDED.is_synthetic,
    synthetic_reason = EXCLUDED.synthetic_reason,
    provenance_verified_at = EXCLUDED.provenance_verified_at;

INSERT INTO artifact_approvals (
    id,
    artifact_id,
    state,
    previous_state,
    approved_by,
    approved_at,
    validation_run_id,
    validation_passed,
    validation_report_uri,
    review_notes,
    reviewer,
    reviewed_at,
    state_changed_by,
    state_changed_at,
    state_change_reason,
    created_at,
    updated_at
)
VALUES (
    '44444444-4444-4444-4444-444444444444',
    '33333333-3333-3333-3333-333333333333',
    'APPROVED'::artifact_approval_state,
    'REVIEWED'::artifact_approval_state,
    'bootstrap',
    '2026-03-11 10:20:00+00',
    '22222222-2222-2222-2222-222222222222',
    TRUE,
    '/reports/artifacts/rsi-momentum-2026-03-11.md',
    'Bootstrap approved artifact for authenticated paper-trading startup',
    'bootstrap',
    '2026-03-11 10:20:00+00',
    'bootstrap',
    '2026-03-11 10:20:00+00',
    'bootstrap approved artifact for trader runtime',
    '2026-03-11 10:20:00+00',
    '2026-03-11 10:20:00+00'
)
ON CONFLICT (artifact_id) DO UPDATE SET
    state = EXCLUDED.state,
    previous_state = EXCLUDED.previous_state,
    approved_by = EXCLUDED.approved_by,
    approved_at = EXCLUDED.approved_at,
    validation_run_id = EXCLUDED.validation_run_id,
    validation_passed = EXCLUDED.validation_passed,
    validation_report_uri = EXCLUDED.validation_report_uri,
    review_notes = EXCLUDED.review_notes,
    reviewer = EXCLUDED.reviewer,
    reviewed_at = EXCLUDED.reviewed_at,
    state_changed_by = EXCLUDED.state_changed_by,
    state_changed_at = EXCLUDED.state_changed_at,
    state_change_reason = EXCLUDED.state_change_reason,
    updated_at = EXCLUDED.updated_at;
