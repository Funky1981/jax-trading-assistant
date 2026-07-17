# Real Candidate Proof

- Status: candidate_produced
- Generated: 2026-07-17T13:49:49.5039648Z
- Candidates created: 1
- Blocked: 0
- Skipped: 3
- Paper tickets created: 0

## Promotion outcomes

- XLE: skipped / no_enabled_strategy_instance - No compatible enabled ETF strategy instance is configured for XLE. (candidate: none)
- SPY: skipped / no_enabled_strategy_instance - No compatible enabled ETF strategy instance is configured for SPY. (candidate: none)
- GLD: skipped / no_enabled_strategy_instance - No compatible enabled ETF strategy instance is configured for GLD. (candidate: none)
- QQQ: promoted / ready_for_approval_review - Risk review returned ready_for_approval_review; next phase is approval_review. (candidate: 8d2e24c4-9d20-42d0-9a63-78857fca3083)

## Candidates

### QQQ - 8d2e24c4-9d20-42d0-9a63-78857fca3083

- inbox_row_id: 78066d9e-7be7-47b4-b12f-ecca3e0da846
- normalized_event_id: b16d6ce7-0385-4cea-b728-8ee957a831b2
- input_source: world-monitor
- setup_type: sector_news_momentum
- catalyst_summary: Operator-generated local proof input for the real World Monitor normalization and paper-candidate pipeline.
- invalidation_reason: QQQ trades at or below the candidate stop level 490.00, invalidating the confirmed sector-news momentum setup.
- strategy_instance_id: 91f58544-82bc-430d-87f6-4248b8ecefa8
- strategy_id: etf_news_sector_momentum_v1
- raw_source_ref: event_raw:4b9a3d47-a6f4-46c2-8d25-28077709b12f
- source_payload_ref: world_monitor_research_inbox:78066d9e-7be7-47b4-b12f-ecca3e0da846
- decision_log_ref: event_normalized:b16d6ce7-0385-4cea-b728-8ee957a831b2
- entry: 500
- stop: 490
- target: 520
- direction: long
- candidate_status: awaiting_approval
- evidence_status: sufficient
- overall_evidence_score: 0.85
- gate_status: ready_for_risk_review
- trust_gate_ready: True
- trust_gate_next_phase: risk_review
- risk_review_attempted: True
- risk_status: ready_for_approval_review
- risk_ready: True
- risk_evaluated_at: 2026-07-17T13:49:49.231565827Z
- risk_entry_price: 500
- risk_stop_loss_price: 490
- risk_target_price: 520
- stop_distance: 10
- slippage_allowance: 0
- slippage_source: absent_interpreted_as_zero_by_existing_risk_engine
- slippage_adjusted_stop_distance: 10
- account_equity: 10000
- account_equity_source: proof risk-model assumption
- maximum_risk_percentage: 0.01
- maximum_allowed_loss: 100
- position_size: 10
- position_notional: 5000
- maximum_normal_loss: 100
- maximum_slippage_adjusted_loss: 100
- reward_amount: 200
- reward_risk_ratio: 2
- minimum_required_reward_risk: 2
- requested_leverage: 1
- maximum_leverage: 1
- risk_reject_reasons: (none)
- risk_warning_reasons: (none)
- risk_next_required_phase: approval_review
- approval_eligibility: True
- approval_status: approval_review_ready
- human_approval_decision_count: 0
- paper_ticket_status: not_created
- missing_fields: (none)
- validation_reject_reasons: (none)
- reject_reasons: (none)
- warning_reasons: (none)

## Safety

- Execution instructions created: 0
- Unsafe paper tickets: 0
- Leveraged candidates: 0
- Runtime mode: paper
- Allow live trading: False
- Execution worker enabled: False
