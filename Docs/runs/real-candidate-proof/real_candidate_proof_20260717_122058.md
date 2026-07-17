# Real Candidate Proof

- Status: candidate_produced
- Generated: 2026-07-17T11:20:58.7693451Z
- Candidates created: 1
- Blocked: 0
- Skipped: 3
- Paper tickets created: 0

## Promotion outcomes

- QQQ: promoted / ready_for_approval_review - Risk review returned ready_for_approval_review; next phase is approval_review. (candidate: 605f1bb7-b950-4007-8b9a-d7c4545439bf)
- XLE: skipped / no_enabled_strategy_instance - No compatible enabled ETF strategy instance is configured for XLE. (candidate: none)
- SPY: skipped / no_enabled_strategy_instance - No compatible enabled ETF strategy instance is configured for SPY. (candidate: none)
- GLD: skipped / no_enabled_strategy_instance - No compatible enabled ETF strategy instance is configured for GLD. (candidate: none)

## Candidates

### QQQ - 605f1bb7-b950-4007-8b9a-d7c4545439bf

- inbox_row_id: f630c1d0-1d7d-4f1a-8fa1-82eb55dd0d4a
- normalized_event_id: c26e94ff-8051-4f42-ad2a-72860b16155f
- input_source: world-monitor
- setup_type: sector_news_momentum
- catalyst_summary: Operator-generated local proof input for the real World Monitor normalization and paper-candidate pipeline.
- invalidation_reason: QQQ trades at or below the candidate stop level 490.00, invalidating the confirmed sector-news momentum setup.
- strategy_instance_id: 91f58544-82bc-430d-87f6-4248b8ecefa8
- strategy_id: etf_news_sector_momentum_v1
- raw_source_ref: event_raw:d91a7976-600f-40d4-b352-442351716943
- source_payload_ref: world_monitor_research_inbox:f630c1d0-1d7d-4f1a-8fa1-82eb55dd0d4a
- decision_log_ref: event_normalized:c26e94ff-8051-4f42-ad2a-72860b16155f
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
- risk_evaluated_at: 2026-07-17T11:20:58.55318097Z
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
