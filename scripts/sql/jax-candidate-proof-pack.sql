-- READ-ONLY JAX PROOF QUERIES
-- Safe to run section by section in pgAdmin.
-- Do not add INSERT, UPDATE, DELETE, TRUNCATE, DROP or ALTER statements.
--
-- Connection: localhost:5433 / database jax / user jax
--
-- Usage:
--   1. Run section 0 once in the pgAdmin Query Tool session.
--   2. Leave the candidate UUID blank to inspect the latest World Monitor
--      candidate, or replace it with a specific candidate UUID.
--   3. Run any later section independently in the same session.
--
-- This pack never approves a candidate, creates a paper ticket, creates an
-- execution instruction, or changes application data. set_config below only
-- sets a custom parameter in the current PostgreSQL session.

-- ============================================================================
-- 0. SESSION PARAMETER AND DATABASE IDENTITY
-- ============================================================================

SELECT set_config(
    'jax.proof_candidate_id',
    '', -- Example: 605f1bb7-b950-4007-8b9a-d7c4545439bf
    false
) AS configured_candidate_id;

SELECT
    current_database() AS database_name,
    current_user AS database_user,
    inet_server_addr() AS server_address,
    inet_server_port() AS server_port,
    current_setting('jax.proof_candidate_id', true) AS requested_candidate_id,
    now() AS inspected_at;

SELECT version AS schema_version, dirty AS migration_dirty
FROM schema_migrations
ORDER BY version DESC
LIMIT 1;

-- ============================================================================
-- 1. SELECTED CANDIDATE
-- Blank parameter = latest candidate linked to a World Monitor inbox row.
-- ============================================================================

WITH parameter AS (
    SELECT NULLIF(current_setting('jax.proof_candidate_id', true), '')::uuid AS candidate_id
), selected AS (
    SELECT ct.*
    FROM candidate_trades ct
    JOIN world_monitor_research_inbox w ON w.candidate_id = ct.id
    CROSS JOIN parameter p
    WHERE p.candidate_id IS NULL OR ct.id = p.candidate_id
    ORDER BY ct.created_at DESC
    LIMIT 1
)
SELECT
    s.id AS candidate_id,
    s.symbol,
    s.status AS candidate_lifecycle_status,
    s.source,
    s.strategy_id,
    s.strategy_instance_id,
    s.created_at,
    s.expires_at,
    s.expires_at IS NOT NULL AND s.expires_at < now() AS candidate_expired,
    s.gate_status,
    s.risk_status,
    s.approval_status
FROM selected s;

-- ============================================================================
-- 2. WORLD MONITOR INPUT, RAW EVENT, AND NORMALIZED EVENT PROVENANCE
-- is_synthetic and synthetic_reason are authoritative provenance fields.
-- The source-event prefix is shown only as an additional proof-input hint.
-- ============================================================================

WITH parameter AS (
    SELECT NULLIF(current_setting('jax.proof_candidate_id', true), '')::uuid AS candidate_id
), selected AS (
    SELECT ct.id
    FROM candidate_trades ct
    JOIN world_monitor_research_inbox w ON w.candidate_id = ct.id
    CROSS JOIN parameter p
    WHERE p.candidate_id IS NULL OR ct.id = p.candidate_id
    ORDER BY ct.created_at DESC
    LIMIT 1
)
SELECT
    w.id AS inbox_row_id,
    w.source,
    w.source_event_id,
    w.world_monitor_event_id,
    w.status AS inbox_status,
    w.event_type,
    w.headline,
    w.summary AS inbox_summary,
    w.event_time,
    w.received_at,
    w.source_urls,
    w.source_count,
    w.possible_affected_etfs,
    w.confidence,
    w.mapping_reason,
    w.normalized_event_id,
    er.id AS raw_event_id,
    er.data_source_type AS raw_data_source_type,
    er.source_provider AS raw_source_provider,
    er.is_synthetic AS raw_is_synthetic,
    er.synthetic_reason AS raw_synthetic_reason,
    er.provenance_verified_at AS raw_provenance_verified_at,
    en.canonical_key,
    en.event_kind AS normalized_event_kind,
    en.title AS normalized_title,
    en.summary AS normalized_summary,
    en.confidence AS normalized_confidence,
    en.attributes AS normalized_attributes,
    en.data_source_type AS normalized_data_source_type,
    en.source_provider AS normalized_source_provider,
    en.is_synthetic AS normalized_is_synthetic,
    en.synthetic_reason AS normalized_synthetic_reason,
    en.provenance_verified_at AS normalized_provenance_verified_at,
    CASE
        WHEN COALESCE(er.is_synthetic, false) OR COALESCE(en.is_synthetic, false)
            THEN 'synthetic_proof_input'
        WHEN w.source ILIKE '%proof%' OR w.source_event_id LIKE 'real-qqq-proof-%'
            THEN 'synthetic_proof_input_provenance_flag_mismatch'
        WHEN COALESCE(er.data_source_type, en.data_source_type) = 'real'
            THEN 'live_or_real_input'
        ELSE 'input_provenance_unclassified'
    END AS input_classification
FROM selected s
JOIN world_monitor_research_inbox w ON w.candidate_id = s.id
LEFT JOIN event_normalized en ON en.id = w.normalized_event_id
LEFT JOIN event_raw er ON er.id = en.raw_event_id;

-- ============================================================================
-- 3. MATCHED STRATEGY INSTANCE AND CANDIDATE PROVENANCE REFERENCES
-- ============================================================================

WITH parameter AS (
    SELECT NULLIF(current_setting('jax.proof_candidate_id', true), '')::uuid AS candidate_id
), selected AS (
    SELECT ct.*
    FROM candidate_trades ct
    JOIN world_monitor_research_inbox w ON w.candidate_id = ct.id
    CROSS JOIN parameter p
    WHERE p.candidate_id IS NULL OR ct.id = p.candidate_id
    ORDER BY ct.created_at DESC
    LIMIT 1
)
SELECT
    s.id AS candidate_id,
    s.symbol,
    s.signal_type,
    s.strategy_instance_id,
    si.name AS strategy_instance_name,
    si.enabled AS strategy_instance_enabled,
    si.strategy_type_id,
    si.strategy_id AS instance_strategy_id,
    si.config AS strategy_instance_config,
    s.strategy_id AS candidate_strategy_id,
    s.signal_id,
    s.raw_source_ref,
    s.source_payload_ref,
    s.decision_log_ref,
    s.data_provenance,
    s.metadata -> 'worldMonitor' AS world_monitor_metadata,
    s.metadata -> 'chartConfirmation' AS chart_confirmation_metadata
FROM selected s
LEFT JOIN strategy_instances si ON si.id = s.strategy_instance_id;

-- ============================================================================
-- 4. STRUCTURED CANDIDATE FIELD COMPLETENESS
-- One row per required/review field makes missing values obvious in pgAdmin.
-- ============================================================================

WITH parameter AS (
    SELECT NULLIF(current_setting('jax.proof_candidate_id', true), '')::uuid AS candidate_id
), selected AS (
    SELECT ct.*
    FROM candidate_trades ct
    JOIN world_monitor_research_inbox w ON w.candidate_id = ct.id
    CROSS JOIN parameter p
    WHERE p.candidate_id IS NULL OR ct.id = p.candidate_id
    ORDER BY ct.created_at DESC
    LIMIT 1
)
SELECT
    s.id AS candidate_id,
    f.field_name,
    f.field_value,
    f.populated,
    CASE WHEN f.populated THEN 'PASS' ELSE 'MISSING' END AS proof_result
FROM selected s
CROSS JOIN LATERAL (
    VALUES
        ('symbol', s.symbol, NULLIF(btrim(s.symbol), '') IS NOT NULL),
        ('source', s.source, NULLIF(btrim(s.source), '') IS NOT NULL),
        ('instrument_type', s.instrument_type, NULLIF(btrim(s.instrument_type), '') IS NOT NULL),
        ('setup_type', s.setup_type, NULLIF(btrim(s.setup_type), '') IS NOT NULL),
        ('direction', s.direction, NULLIF(btrim(s.direction), '') IS NOT NULL),
        ('time_horizon', s.time_horizon, NULLIF(btrim(s.time_horizon), '') IS NOT NULL),
        ('catalyst_summary', s.catalyst_summary, NULLIF(btrim(s.catalyst_summary), '') IS NOT NULL),
        ('invalidation_reason', s.invalidation_reason, NULLIF(btrim(s.invalidation_reason), '') IS NOT NULL),
        ('entry_price', s.entry_price::text, s.entry_price > 0),
        ('stop_loss', s.stop_loss::text, s.stop_loss > 0),
        ('take_profit', s.take_profit::text, s.take_profit > 0),
        ('strategy_instance_id', s.strategy_instance_id::text, s.strategy_instance_id IS NOT NULL),
        ('strategy_id', s.strategy_id, NULLIF(btrim(s.strategy_id), '') IS NOT NULL),
        ('raw_source_ref', s.raw_source_ref, NULLIF(btrim(s.raw_source_ref), '') IS NOT NULL),
        ('source_payload_ref', s.source_payload_ref, NULLIF(btrim(s.source_payload_ref), '') IS NOT NULL),
        ('decision_log_ref', s.decision_log_ref, NULLIF(btrim(s.decision_log_ref), '') IS NOT NULL)
) AS f(field_name, field_value, populated)
ORDER BY f.field_name;

-- Compact structured-completeness verdict.
WITH parameter AS (
    SELECT NULLIF(current_setting('jax.proof_candidate_id', true), '')::uuid AS candidate_id
), selected AS (
    SELECT ct.*
    FROM candidate_trades ct
    JOIN world_monitor_research_inbox w ON w.candidate_id = ct.id
    CROSS JOIN parameter p
    WHERE p.candidate_id IS NULL OR ct.id = p.candidate_id
    ORDER BY ct.created_at DESC
    LIMIT 1
)
SELECT
    s.id AS candidate_id,
    NULLIF(btrim(s.symbol), '') IS NOT NULL
        AND NULLIF(btrim(s.setup_type), '') IS NOT NULL
        AND NULLIF(btrim(s.direction), '') IS NOT NULL
        AND NULLIF(btrim(s.catalyst_summary), '') IS NOT NULL
        AND NULLIF(btrim(s.invalidation_reason), '') IS NOT NULL
        AND s.entry_price > 0
        AND s.stop_loss > 0 AS structurally_complete,
    s.entry_price,
    s.stop_loss,
    s.take_profit,
    s.reject_reasons,
    s.blocked_reason_code,
    s.block_reason
FROM selected s;

-- ============================================================================
-- 5. GENUINE PERSISTED EVIDENCE ITEMS
-- ============================================================================

WITH parameter AS (
    SELECT NULLIF(current_setting('jax.proof_candidate_id', true), '')::uuid AS candidate_id
), selected AS (
    SELECT ct.id
    FROM candidate_trades ct
    JOIN world_monitor_research_inbox w ON w.candidate_id = ct.id
    CROSS JOIN parameter p
    WHERE p.candidate_id IS NULL OR ct.id = p.candidate_id
    ORDER BY ct.created_at DESC
    LIMIT 1
)
SELECT
    ei.candidate_id,
    ei.evidence_id,
    ei.evidence_kind,
    ei.source_type,
    ei.source_ref,
    ei.observed_at,
    ei.summary,
    ei.supports_candidate,
    ei.contradicts_candidate,
    ei.confidence,
    ei.impact_score,
    ei.quality_score,
    ei.freshness_status,
    ei.notes,
    ei.created_at
FROM selected s
JOIN candidate_evidence_items ei ON ei.candidate_id = s.id
ORDER BY ei.observed_at, ei.created_at;

-- ============================================================================
-- 6. EVIDENCE SCORE HISTORY AND LATEST TRUST-GATE INPUT
-- ============================================================================

WITH parameter AS (
    SELECT NULLIF(current_setting('jax.proof_candidate_id', true), '')::uuid AS candidate_id
), selected AS (
    SELECT ct.id
    FROM candidate_trades ct
    JOIN world_monitor_research_inbox w ON w.candidate_id = ct.id
    CROSS JOIN parameter p
    WHERE p.candidate_id IS NULL OR ct.id = p.candidate_id
    ORDER BY ct.created_at DESC
    LIMIT 1
)
SELECT
    es.*,
    es.evidence_status = 'sufficient'
        AND es.evidence_ready
        AND es.evidence_gate_ready AS evidence_passed
FROM selected s
JOIN candidate_evidence_scores es ON es.candidate_id = s.id
ORDER BY es.scored_at DESC;

-- Latest evidence score plus persisted candidate trust-gate result.
WITH parameter AS (
    SELECT NULLIF(current_setting('jax.proof_candidate_id', true), '')::uuid AS candidate_id
), selected AS (
    SELECT ct.*
    FROM candidate_trades ct
    JOIN world_monitor_research_inbox w ON w.candidate_id = ct.id
    CROSS JOIN parameter p
    WHERE p.candidate_id IS NULL OR ct.id = p.candidate_id
    ORDER BY ct.created_at DESC
    LIMIT 1
), latest_score AS (
    SELECT DISTINCT ON (es.candidate_id) es.*
    FROM candidate_evidence_scores es
    JOIN selected s ON s.id = es.candidate_id
    ORDER BY es.candidate_id, es.scored_at DESC
)
SELECT
    s.id AS candidate_id,
    ls.evidence_status,
    ls.overall_evidence_score,
    ls.evidence_ready,
    ls.evidence_gate_ready,
    s.gate_status AS persisted_gate_status,
    s.gate_status = 'ready_for_risk_review' AS trust_gate_ready,
    CASE
        WHEN s.gate_status = 'ready_for_risk_review' THEN 'risk_review'
        WHEN s.gate_status IN ('evidence_missing', 'evidence_weak', 'evidence_mixed', 'evidence_stale') THEN 'evidence_review'
        WHEN s.gate_status = 'incomplete' THEN 'candidate_repair'
        ELSE 'stop_or_investigate_gate_status'
    END AS trust_gate_next_phase,
    ls.broker_execution_allowed,
    ls.execution_instruction_created,
    ls.approval_granted,
    ls.scored_at
FROM selected s
LEFT JOIN latest_score ls ON ls.candidate_id = s.id;

-- ============================================================================
-- 7. RISK REVIEW RESULT AND CONFIGURATION SOURCES
-- metadata.riskReview is the complete current risk snapshot.
-- ============================================================================

WITH parameter AS (
    SELECT NULLIF(current_setting('jax.proof_candidate_id', true), '')::uuid AS candidate_id
), selected AS (
    SELECT ct.*
    FROM candidate_trades ct
    JOIN world_monitor_research_inbox w ON w.candidate_id = ct.id
    CROSS JOIN parameter p
    WHERE p.candidate_id IS NULL OR ct.id = p.candidate_id
    ORDER BY ct.created_at DESC
    LIMIT 1
)
SELECT
    s.id AS candidate_id,
    s.metadata -> 'riskReview' -> 'result' IS NOT NULL AS risk_review_ran,
    s.risk_status AS persisted_risk_status,
    (s.metadata -> 'riskReview' -> 'result' ->> 'evaluatedAt')::timestamptz AS evaluated_at,
    COALESCE((s.metadata -> 'riskReview' -> 'result' ->> 'riskReady')::boolean, false) AS risk_ready,
    (s.metadata -> 'riskReview' -> 'result' ->> 'entryPrice')::numeric AS entry_price,
    (s.metadata -> 'riskReview' -> 'result' ->> 'stopLossPrice')::numeric AS stop_loss_price,
    (s.metadata -> 'riskReview' -> 'result' ->> 'targetPrice')::numeric AS target_price,
    (s.metadata -> 'riskReview' -> 'result' ->> 'stopDistance')::numeric AS stop_distance,
    (s.metadata -> 'riskReview' -> 'result' ->> 'slippageAllowance')::numeric AS slippage_allowance,
    s.metadata -> 'riskReview' ->> 'slippageSource' AS slippage_source,
    (s.metadata -> 'riskReview' -> 'result' ->> 'slippageAdjustedStopDistance')::numeric AS slippage_adjusted_stop_distance,
    (s.metadata -> 'riskReview' -> 'result' ->> 'accountEquity')::numeric AS account_equity,
    s.metadata -> 'riskReview' ->> 'accountEquitySource' AS account_equity_source,
    s.metadata -> 'riskReview' ->> 'riskPolicySource' AS risk_policy_source,
    (s.metadata -> 'riskReview' -> 'result' ->> 'maxRiskPercent')::numeric AS maximum_risk_percentage,
    (s.metadata -> 'riskReview' -> 'result' ->> 'maxAllowedLoss')::numeric AS maximum_allowed_loss,
    (s.metadata -> 'riskReview' -> 'result' ->> 'positionSize')::numeric AS theoretical_position_size,
    (s.metadata -> 'riskReview' ->> 'positionNotional')::numeric AS theoretical_position_notional,
    (s.metadata -> 'riskReview' -> 'result' ->> 'maxNormalLoss')::numeric AS maximum_normal_loss,
    (s.metadata -> 'riskReview' -> 'result' ->> 'maxSlippageAdjustedLoss')::numeric AS maximum_slippage_adjusted_loss,
    (s.metadata -> 'riskReview' -> 'result' ->> 'rewardAmount')::numeric AS reward_amount,
    (s.metadata -> 'riskReview' -> 'result' ->> 'rewardRiskRatio')::numeric AS reward_risk_ratio,
    (s.metadata -> 'riskReview' -> 'result' ->> 'minRewardRiskRatio')::numeric AS minimum_required_reward_risk,
    (s.metadata -> 'riskReview' -> 'result' ->> 'requestedLeverage')::numeric AS requested_leverage,
    (s.metadata -> 'riskReview' -> 'result' ->> 'maxLeverage')::numeric AS maximum_leverage,
    s.metadata -> 'riskReview' -> 'result' -> 'rejectReasons' AS risk_reject_reasons,
    s.metadata -> 'riskReview' -> 'result' -> 'warningReasons' AS risk_warning_reasons,
    s.metadata -> 'riskReview' -> 'result' ->> 'nextRequiredPhase' AS risk_next_required_phase,
    COALESCE((s.metadata -> 'riskReview' -> 'result' ->> 'approvalGranted')::boolean, false) AS approval_granted_by_risk,
    COALESCE((s.metadata -> 'riskReview' -> 'result' ->> 'brokerExecutionAllowed')::boolean, false) AS broker_execution_allowed_by_risk,
    COALESCE((s.metadata -> 'riskReview' -> 'result' ->> 'executionInstructionCreated')::boolean, false) AS execution_instruction_created_by_risk
FROM selected s;

-- ============================================================================
-- 8. COMPUTED APPROVAL ELIGIBILITY VERSUS HUMAN APPROVAL DECISIONS
-- Approval eligibility is not a human approval decision.
-- ============================================================================

WITH parameter AS (
    SELECT NULLIF(current_setting('jax.proof_candidate_id', true), '')::uuid AS candidate_id
), selected AS (
    SELECT ct.*
    FROM candidate_trades ct
    JOIN world_monitor_research_inbox w ON w.candidate_id = ct.id
    CROSS JOIN parameter p
    WHERE p.candidate_id IS NULL OR ct.id = p.candidate_id
    ORDER BY ct.created_at DESC
    LIMIT 1
), latest_score AS (
    SELECT DISTINCT ON (es.candidate_id) es.*
    FROM candidate_evidence_scores es
    JOIN selected s ON s.id = es.candidate_id
    ORDER BY es.candidate_id, es.scored_at DESC
), approval_counts AS (
    SELECT ca.candidate_id,
           count(*) AS decision_count,
           count(*) FILTER (WHERE ca.decision = 'approved') AS approved_decision_count,
           max(ca.decided_at) AS latest_decision_at
    FROM candidate_approvals ca
    JOIN selected s ON s.id = ca.candidate_id
    GROUP BY ca.candidate_id
)
SELECT
    s.id AS candidate_id,
    s.status AS candidate_lifecycle_status,
    s.approval_status AS persisted_approval_status,
    COALESCE((s.metadata -> 'riskReview' -> 'approvalEligibility' ->> 'approvalEligible')::boolean, false) AS persisted_approval_eligible,
    s.status = 'awaiting_approval'
        AND (s.expires_at IS NULL OR s.expires_at >= now())
        AND ls.evidence_status = 'sufficient'
        AND ls.evidence_ready
        AND ls.evidence_gate_ready
        AND s.gate_status = 'ready_for_risk_review'
        AND s.risk_status = 'ready_for_approval_review'
        AND s.approval_status = 'approval_review_ready'
        AND COALESCE((s.metadata -> 'riskReview' -> 'result' ->> 'riskReady')::boolean, false)
        AND COALESCE((s.metadata -> 'riskReview' -> 'result' ->> 'approvalGranted')::boolean, false) = false
        AND COALESCE((s.metadata -> 'riskReview' -> 'result' ->> 'brokerExecutionAllowed')::boolean, false) = false
        AND COALESCE((s.metadata -> 'riskReview' -> 'result' ->> 'executionInstructionCreated')::boolean, false) = false
        AS currently_approval_eligible,
    COALESCE(ac.decision_count, 0) AS human_decision_count,
    COALESCE(ac.approved_decision_count, 0) AS human_approved_decision_count,
    COALESCE(ac.approved_decision_count, 0) > 0 AS human_approved,
    ac.latest_decision_at,
    s.expires_at,
    s.expires_at IS NOT NULL AND s.expires_at < now() AS candidate_expired
FROM selected s
LEFT JOIN latest_score ls ON ls.candidate_id = s.id
LEFT JOIN approval_counts ac ON ac.candidate_id = s.id;

-- Full human decision history, if any.
WITH parameter AS (
    SELECT NULLIF(current_setting('jax.proof_candidate_id', true), '')::uuid AS candidate_id
), selected AS (
    SELECT ct.id
    FROM candidate_trades ct
    JOIN world_monitor_research_inbox w ON w.candidate_id = ct.id
    CROSS JOIN parameter p
    WHERE p.candidate_id IS NULL OR ct.id = p.candidate_id
    ORDER BY ct.created_at DESC
    LIMIT 1
)
SELECT ca.*
FROM selected s
JOIN candidate_approvals ca ON ca.candidate_id = s.id
ORDER BY ca.decided_at, ca.created_at;

-- ============================================================================
-- 9. PAPER TICKET AND EXECUTION-INSTRUCTION RECORDS
-- Empty result sets are expected before their explicit lifecycle boundaries.
-- ============================================================================

WITH parameter AS (
    SELECT NULLIF(current_setting('jax.proof_candidate_id', true), '')::uuid AS candidate_id
), selected AS (
    SELECT ct.id
    FROM candidate_trades ct
    JOIN world_monitor_research_inbox w ON w.candidate_id = ct.id
    CROSS JOIN parameter p
    WHERE p.candidate_id IS NULL OR ct.id = p.candidate_id
    ORDER BY ct.created_at DESC
    LIMIT 1
)
SELECT pt.*
FROM selected s
JOIN candidate_paper_tickets pt ON pt.candidate_id = s.id
ORDER BY pt.created_at;

WITH parameter AS (
    SELECT NULLIF(current_setting('jax.proof_candidate_id', true), '')::uuid AS candidate_id
), selected AS (
    SELECT ct.id
    FROM candidate_trades ct
    JOIN world_monitor_research_inbox w ON w.candidate_id = ct.id
    CROSS JOIN parameter p
    WHERE p.candidate_id IS NULL OR ct.id = p.candidate_id
    ORDER BY ct.created_at DESC
    LIMIT 1
)
SELECT ei.*
FROM selected s
JOIN execution_instructions ei ON ei.candidate_id = s.id
ORDER BY ei.created_at;

-- ============================================================================
-- 10. PAPER-ONLY / NO-LEVERAGE / NO-EXECUTION SAFETY VERDICT
-- safety_pass must be true. Any execution instruction is a proof failure.
-- ============================================================================

WITH parameter AS (
    SELECT NULLIF(current_setting('jax.proof_candidate_id', true), '')::uuid AS candidate_id
), selected AS (
    SELECT ct.*
    FROM candidate_trades ct
    JOIN world_monitor_research_inbox w ON w.candidate_id = ct.id
    CROSS JOIN parameter p
    WHERE p.candidate_id IS NULL OR ct.id = p.candidate_id
    ORDER BY ct.created_at DESC
    LIMIT 1
), latest_score AS (
    SELECT DISTINCT ON (es.candidate_id) es.*
    FROM candidate_evidence_scores es
    JOIN selected s ON s.id = es.candidate_id
    ORDER BY es.candidate_id, es.scored_at DESC
), counts AS (
    SELECT
        s.id AS candidate_id,
        count(DISTINCT ca.id) AS human_decision_count,
        count(DISTINCT pt.id) AS paper_ticket_count,
        count(DISTINCT ei.id) AS execution_instruction_count
    FROM selected s
    LEFT JOIN candidate_approvals ca ON ca.candidate_id = s.id
    LEFT JOIN candidate_paper_tickets pt ON pt.candidate_id = s.id
    LEFT JOIN execution_instructions ei ON ei.candidate_id = s.id
    GROUP BY s.id
), ticket_safety AS (
    SELECT
        s.id AS candidate_id,
        COALESCE(bool_and(
            pt.paper_only
            AND NOT pt.broker_execution_allowed
            AND NOT pt.execution_instruction_created
            AND NOT pt.live_trading_allowed
            AND NOT pt.leverage_allowed
        ), true) AS all_tickets_safe
    FROM selected s
    LEFT JOIN candidate_paper_tickets pt ON pt.candidate_id = s.id
    GROUP BY s.id
)
SELECT
    s.id AS candidate_id,
    COALESCE((s.metadata ->> 'paperOnly')::boolean, false) AS candidate_paper_only,
    COALESCE((s.metadata -> 'riskReview' -> 'result' ->> 'requestedLeverage')::numeric, 1) AS requested_leverage,
    COALESCE((s.metadata -> 'riskReview' -> 'result' ->> 'maxLeverage')::numeric, 1) AS maximum_leverage,
    COALESCE(ls.approval_granted, false) AS evidence_approval_granted,
    COALESCE(ls.broker_execution_allowed, false) AS evidence_broker_execution_allowed,
    COALESCE(ls.execution_instruction_created, false) AS evidence_execution_instruction_created,
    COALESCE((s.metadata -> 'riskReview' -> 'result' ->> 'approvalGranted')::boolean, false) AS risk_approval_granted,
    COALESCE((s.metadata -> 'riskReview' -> 'result' ->> 'brokerExecutionAllowed')::boolean, false) AS risk_broker_execution_allowed,
    COALESCE((s.metadata -> 'riskReview' -> 'result' ->> 'executionInstructionCreated')::boolean, false) AS risk_execution_instruction_created,
    c.human_decision_count,
    c.paper_ticket_count,
    c.execution_instruction_count,
    ts.all_tickets_safe,
    COALESCE((s.metadata ->> 'paperOnly')::boolean, false)
        AND COALESCE((s.metadata -> 'riskReview' -> 'result' ->> 'requestedLeverage')::numeric, 1) <= 1
        AND COALESCE((s.metadata -> 'riskReview' -> 'result' ->> 'maxLeverage')::numeric, 1) <= 1
        AND NOT COALESCE(ls.approval_granted, false)
        AND NOT COALESCE(ls.broker_execution_allowed, false)
        AND NOT COALESCE(ls.execution_instruction_created, false)
        AND NOT COALESCE((s.metadata -> 'riskReview' -> 'result' ->> 'approvalGranted')::boolean, false)
        AND NOT COALESCE((s.metadata -> 'riskReview' -> 'result' ->> 'brokerExecutionAllowed')::boolean, false)
        AND NOT COALESCE((s.metadata -> 'riskReview' -> 'result' ->> 'executionInstructionCreated')::boolean, false)
        AND c.execution_instruction_count = 0
        AND ts.all_tickets_safe AS safety_pass
FROM selected s
LEFT JOIN latest_score ls ON ls.candidate_id = s.id
JOIN counts c ON c.candidate_id = s.id
JOIN ticket_safety ts ON ts.candidate_id = s.id;

-- Database-wide inventory is deliberately separate from the selected-candidate
-- verdict. Historical or unrelated records must not be mistaken for records
-- created by the selected proof candidate.
SELECT
    (SELECT count(*) FROM execution_instructions) AS all_execution_instruction_count,
    (
        SELECT count(*)
        FROM execution_instructions ei
        JOIN candidate_trades ct ON ct.id = ei.candidate_id
        WHERE ct.source = 'world-monitor'
    ) AS world_monitor_execution_instruction_count,
    (
        SELECT count(*)
        FROM candidate_paper_tickets pt
        WHERE NOT pt.paper_only
           OR pt.broker_execution_allowed
           OR pt.execution_instruction_created
           OR pt.live_trading_allowed
           OR pt.leverage_allowed
    ) AS unsafe_paper_ticket_count,
    (
        SELECT count(*)
        FROM candidate_trades ct
        WHERE ct.source = 'world-monitor'
          AND (
              COALESCE((ct.metadata -> 'riskReview' -> 'result' ->> 'requestedLeverage')::numeric, 1) > 1
              OR COALESCE((ct.metadata -> 'riskReview' -> 'result' ->> 'maxLeverage')::numeric, 1) > 1
          )
    ) AS leveraged_world_monitor_candidate_count;

-- Audit any execution instructions present anywhere in the database.
SELECT
    ei.id AS execution_instruction_id,
    ei.candidate_id,
    ct.source AS candidate_source,
    ct.symbol,
    ct.status AS candidate_lifecycle_status,
    ei.status AS execution_instruction_status,
    ei.broker_order_id,
    ei.trade_id,
    ei.created_at
FROM execution_instructions ei
LEFT JOIN candidate_trades ct ON ct.id = ei.candidate_id
ORDER BY ei.created_at, ei.id;

-- ============================================================================
-- 11. CANDIDATE LIFECYCLE AUDIT EVENTS
-- ============================================================================

WITH parameter AS (
    SELECT NULLIF(current_setting('jax.proof_candidate_id', true), '')::uuid AS candidate_id
), selected AS (
    SELECT ct.id
    FROM candidate_trades ct
    JOIN world_monitor_research_inbox w ON w.candidate_id = ct.id
    CROSS JOIN parameter p
    WHERE p.candidate_id IS NULL OR ct.id = p.candidate_id
    ORDER BY ct.created_at DESC
    LIMIT 1
)
SELECT
    ce.candidate_id,
    ce.occurred_at,
    ce.event_type,
    ce.from_status,
    ce.to_status,
    ce.detail
FROM selected s
JOIN candidate_events ce ON ce.candidate_id = s.id
ORDER BY ce.occurred_at, ce.id;

-- ============================================================================
-- 12. ONE-ROW OPERATOR VERDICT AND PRECISE NEXT BLOCKER
-- This is the quickest section to rerun after inspecting the detailed proofs.
-- ============================================================================

WITH parameter AS (
    SELECT NULLIF(current_setting('jax.proof_candidate_id', true), '')::uuid AS candidate_id
), selected AS (
    SELECT ct.*
    FROM candidate_trades ct
    JOIN world_monitor_research_inbox w ON w.candidate_id = ct.id
    CROSS JOIN parameter p
    WHERE p.candidate_id IS NULL OR ct.id = p.candidate_id
    ORDER BY ct.created_at DESC
    LIMIT 1
), latest_score AS (
    SELECT DISTINCT ON (es.candidate_id) es.*
    FROM candidate_evidence_scores es
    JOIN selected s ON s.id = es.candidate_id
    ORDER BY es.candidate_id, es.scored_at DESC
), facts AS (
    SELECT
        s.*,
        ls.evidence_status AS latest_evidence_status,
        COALESCE(ls.evidence_ready, false) AS latest_evidence_ready,
        COALESCE(ls.evidence_gate_ready, false) AS latest_evidence_gate_ready,
        COALESCE((s.metadata -> 'riskReview' -> 'result' ->> 'riskReady')::boolean, false) AS risk_ready,
        COALESCE((s.metadata -> 'riskReview' -> 'approvalEligibility' ->> 'approvalEligible')::boolean, false) AS persisted_approval_eligible,
        (SELECT count(*) FROM candidate_approvals ca WHERE ca.candidate_id = s.id) AS human_decision_count,
        (SELECT count(*) FROM candidate_approvals ca WHERE ca.candidate_id = s.id AND ca.decision = 'approved') AS human_approved_count,
        (SELECT count(*) FROM candidate_paper_tickets pt WHERE pt.candidate_id = s.id) AS paper_ticket_count,
        (SELECT count(*) FROM execution_instructions ei WHERE ei.candidate_id = s.id) AS execution_instruction_count
    FROM selected s
    LEFT JOIN latest_score ls ON ls.candidate_id = s.id
)
SELECT
    f.id AS candidate_id,
    f.symbol,
    f.status AS candidate_lifecycle_status,
    f.latest_evidence_status AS evidence_status,
    f.gate_status,
    f.risk_status,
    f.risk_ready,
    f.approval_status,
    f.persisted_approval_eligible,
    f.human_decision_count,
    f.human_approved_count,
    f.paper_ticket_count,
    f.execution_instruction_count,
    f.expires_at,
    f.expires_at IS NOT NULL AND f.expires_at < now() AS candidate_expired,
    CASE
        WHEN f.execution_instruction_count > 0 THEN 'SAFETY_BREACH_execution_instruction_exists'
        WHEN COALESCE((f.metadata -> 'riskReview' -> 'result' ->> 'requestedLeverage')::numeric, 1) > 1
          OR COALESCE((f.metadata -> 'riskReview' -> 'result' ->> 'maxLeverage')::numeric, 1) > 1
            THEN 'SAFETY_BREACH_leverage_above_1'
        WHEN NOT COALESCE((f.metadata ->> 'paperOnly')::boolean, false)
            THEN 'SAFETY_BREACH_candidate_not_paper_only'
        WHEN f.expires_at IS NOT NULL AND f.expires_at < now() THEN 'candidate_expired_create_fresh_proof_candidate'
        WHEN f.status = 'blocked' THEN COALESCE(f.blocked_reason_code, f.block_reason, 'candidate_blocked')
        WHEN f.latest_evidence_status IS NULL THEN 'latest_evidence_score_missing'
        WHEN f.latest_evidence_status <> 'sufficient' OR NOT f.latest_evidence_ready OR NOT f.latest_evidence_gate_ready
            THEN 'evidence_' || COALESCE(f.latest_evidence_status, 'missing')
        WHEN f.gate_status <> 'ready_for_risk_review' THEN 'trust_gate_' || COALESCE(f.gate_status, 'missing')
        WHEN f.metadata -> 'riskReview' -> 'result' IS NULL THEN 'risk_review_not_run'
        WHEN NOT f.risk_ready THEN COALESCE(f.risk_status, 'risk_not_ready')
        WHEN f.risk_status <> 'ready_for_approval_review' THEN COALESCE(f.risk_status, 'risk_not_ready')
        WHEN f.human_approved_count = 0 THEN 'human_approval_required'
        WHEN f.paper_ticket_count = 0 THEN 'paper_ticket_creation_pending'
        ELSE 'paper_ticket_review_only_no_execution'
    END AS precise_next_blocker_or_phase,
    CASE
        WHEN f.execution_instruction_count = 0
         AND COALESCE((f.metadata ->> 'paperOnly')::boolean, false)
         AND COALESCE((f.metadata -> 'riskReview' -> 'result' ->> 'requestedLeverage')::numeric, 1) <= 1
         AND COALESCE((f.metadata -> 'riskReview' -> 'result' ->> 'maxLeverage')::numeric, 1) <= 1
            THEN true
        ELSE false
    END AS core_safety_intact
FROM facts f;
