-- Codex Packs: Strategy Lab, Research, AI Audit, Trust Gates (ADR-0012 additive rollout)

-- Seed legacy strategy instance used to backfill historical rows.
CREATE TABLE IF NOT EXISTS strategy_instances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    strategy_type_id TEXT NOT NULL,
    strategy_id TEXT,
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    session_timezone TEXT NOT NULL DEFAULT 'America/New_York',
    flatten_by_close_time TEXT NOT NULL DEFAULT '15:55',
    config JSONB NOT NULL DEFAULT '{}'::jsonb,
    config_hash TEXT NOT NULL DEFAULT '',
    artifact_id UUID REFERENCES strategy_artifacts(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_strategy_instances_enabled ON strategy_instances(enabled);
CREATE INDEX IF NOT EXISTS idx_strategy_instances_type ON strategy_instances(strategy_type_id);
CREATE INDEX IF NOT EXISTS idx_strategy_instances_updated_at ON strategy_instances(updated_at DESC);

INSERT INTO strategy_instances (
    id,
    name,
    strategy_type_id,
    strategy_id,
    enabled,
    session_timezone,
    flatten_by_close_time,
    config,
    config_hash
)
VALUES (
    '00000000-0000-0000-0000-000000000001'::uuid,
    'legacy-default',
    'legacy',
    'legacy',
    FALSE,
    'America/New_York',
    '15:55',
    '{"mode":"legacy"}'::jsonb,
    'legacy-default'
)
ON CONFLICT (id) DO NOTHING;

ALTER TABLE strategy_signals
    ADD COLUMN IF NOT EXISTS instance_id UUID;

ALTER TABLE trades
    ADD COLUMN IF NOT EXISTS instance_id UUID;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'fk_strategy_signals_instance'
    ) THEN
        ALTER TABLE strategy_signals
            ADD CONSTRAINT fk_strategy_signals_instance
            FOREIGN KEY (instance_id) REFERENCES strategy_instances(id) ON DELETE SET NULL;
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'fk_trades_instance'
    ) THEN
        ALTER TABLE trades
            ADD CONSTRAINT fk_trades_instance
            FOREIGN KEY (instance_id) REFERENCES strategy_instances(id) ON DELETE SET NULL;
    END IF;
END $$;

UPDATE strategy_signals
SET instance_id = '00000000-0000-0000-0000-000000000001'::uuid
WHERE instance_id IS NULL;

UPDATE trades
SET instance_id = '00000000-0000-0000-0000-000000000001'::uuid
WHERE instance_id IS NULL;

CREATE INDEX IF NOT EXISTS idx_strategy_signals_instance_generated_at
    ON strategy_signals(instance_id, generated_at DESC);
CREATE INDEX IF NOT EXISTS idx_trades_instance_created_at
    ON trades(instance_id, created_at DESC);

CREATE TABLE IF NOT EXISTS backtest_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    external_run_id TEXT UNIQUE,
    instance_id UUID REFERENCES strategy_instances(id) ON DELETE SET NULL,
    strategy_type_id TEXT NOT NULL,
    strategy_config_id TEXT,
    symbols TEXT[] NOT NULL DEFAULT '{}',
    run_from TIMESTAMPTZ NOT NULL,
    run_to TIMESTAMPTZ NOT NULL,
    seed BIGINT NOT NULL,
    dataset_id TEXT,
    status TEXT NOT NULL DEFAULT 'completed',
    stats JSONB NOT NULL DEFAULT '{}'::jsonb,
    config_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    flow_id TEXT,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_backtest_runs_instance_created_at
    ON backtest_runs(instance_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_backtest_runs_status_started_at
    ON backtest_runs(status, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_backtest_runs_external_id
    ON backtest_runs(external_run_id);

CREATE TABLE IF NOT EXISTS backtest_trades (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id UUID NOT NULL REFERENCES backtest_runs(id) ON DELETE CASCADE,
    symbol TEXT NOT NULL,
    side TEXT NOT NULL,
    entry_price DOUBLE PRECISION,
    exit_price DOUBLE PRECISION,
    quantity DOUBLE PRECISION,
    pnl DOUBLE PRECISION,
    pnl_pct DOUBLE PRECISION,
    opened_at TIMESTAMPTZ,
    closed_at TIMESTAMPTZ,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_backtest_trades_run_symbol
    ON backtest_trades(run_id, symbol);

CREATE TABLE IF NOT EXISTS research_projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    owner TEXT,
    status TEXT NOT NULL DEFAULT 'draft',
    base_instance_id UUID REFERENCES strategy_instances(id) ON DELETE SET NULL,
    parameter_grid JSONB NOT NULL DEFAULT '{}'::jsonb,
    train_from TIMESTAMPTZ,
    train_to TIMESTAMPTZ,
    test_from TIMESTAMPTZ,
    test_to TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_research_projects_status_updated_at
    ON research_projects(status, updated_at DESC);

CREATE TABLE IF NOT EXISTS research_project_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES research_projects(id) ON DELETE CASCADE,
    backtest_run_id UUID REFERENCES backtest_runs(id) ON DELETE SET NULL,
    status TEXT NOT NULL DEFAULT 'queued',
    parameters JSONB NOT NULL DEFAULT '{}'::jsonb,
    metrics JSONB NOT NULL DEFAULT '{}'::jsonb,
    rank_score DOUBLE PRECISION,
    lineage JSONB NOT NULL DEFAULT '{}'::jsonb,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_research_project_runs_project_created
    ON research_project_runs(project_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_research_project_runs_rank
    ON research_project_runs(project_id, rank_score DESC);

CREATE TABLE IF NOT EXISTS runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_type TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'running',
    flow_id TEXT,
    source TEXT,
    instance_id UUID REFERENCES strategy_instances(id) ON DELETE SET NULL,
    orchestration_run_id UUID REFERENCES orchestration_runs(id) ON DELETE SET NULL,
    backtest_run_id UUID REFERENCES backtest_runs(id) ON DELETE SET NULL,
    research_project_run_id UUID REFERENCES research_project_runs(id) ON DELETE SET NULL,
    summary JSONB NOT NULL DEFAULT '{}'::jsonb,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_runs_flow_id ON runs(flow_id);
CREATE INDEX IF NOT EXISTS idx_runs_type_created_at ON runs(run_type, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_runs_instance_created_at ON runs(instance_id, created_at DESC);

CREATE TABLE IF NOT EXISTS run_artifacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id UUID NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    artifact_type TEXT NOT NULL,
    artifact_uri TEXT NOT NULL,
    artifact_hash TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_run_artifacts_run_type
    ON run_artifacts(run_id, artifact_type);

CREATE TABLE IF NOT EXISTS ai_decisions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id UUID REFERENCES runs(id) ON DELETE SET NULL,
    flow_id TEXT,
    role TEXT NOT NULL,
    provider TEXT,
    model TEXT,
    prompt JSONB NOT NULL DEFAULT '{}'::jsonb,
    response JSONB NOT NULL DEFAULT '{}'::jsonb,
    schema_valid BOOLEAN NOT NULL DEFAULT FALSE,
    decision TEXT,
    reasoning TEXT,
    rule_trace JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ai_decisions_run_created_at
    ON ai_decisions(run_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_decisions_flow_id
    ON ai_decisions(flow_id);

CREATE TABLE IF NOT EXISTS ai_decision_acceptance (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    decision_id UUID NOT NULL REFERENCES ai_decisions(id) ON DELETE CASCADE,
    accepted BOOLEAN NOT NULL,
    accepted_by TEXT,
    reason TEXT,
    rule_trace JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ai_decision_acceptance_decision
    ON ai_decision_acceptance(decision_id);

CREATE TABLE IF NOT EXISTS test_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id UUID REFERENCES runs(id) ON DELETE SET NULL,
    test_name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued',
    summary JSONB NOT NULL DEFAULT '{}'::jsonb,
    artifact_uri TEXT,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_test_runs_name_created
    ON test_runs(test_name, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_test_runs_status_started
    ON test_runs(status, started_at DESC);

CREATE TABLE IF NOT EXISTS gate_status (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    gate_name TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL,
    last_run_id UUID REFERENCES test_runs(id) ON DELETE SET NULL,
    details JSONB NOT NULL DEFAULT '{}'::jsonb,
    last_run_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_gate_status_status_updated
    ON gate_status(status, updated_at DESC);

CREATE TABLE IF NOT EXISTS order_intents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instance_id UUID REFERENCES strategy_instances(id) ON DELETE SET NULL,
    signal_id UUID REFERENCES strategy_signals(id) ON DELETE SET NULL,
    flow_id TEXT,
    symbol TEXT NOT NULL,
    side TEXT NOT NULL,
    quantity DOUBLE PRECISION NOT NULL,
    order_type TEXT NOT NULL,
    limit_price DOUBLE PRECISION,
    stop_price DOUBLE PRECISION,
    status TEXT NOT NULL DEFAULT 'pending',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_order_intents_instance_created
    ON order_intents(instance_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_order_intents_status_created
    ON order_intents(status, created_at DESC);

CREATE TABLE IF NOT EXISTS fills (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    intent_id UUID REFERENCES order_intents(id) ON DELETE SET NULL,
    trade_id TEXT REFERENCES trades(id) ON DELETE SET NULL,
    broker_order_id TEXT,
    symbol TEXT NOT NULL,
    side TEXT NOT NULL,
    filled_qty DOUBLE PRECISION NOT NULL,
    avg_fill_price DOUBLE PRECISION NOT NULL,
    status TEXT NOT NULL DEFAULT 'filled',
    filled_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_fills_intent_filled_at
    ON fills(intent_id, filled_at DESC);
CREATE INDEX IF NOT EXISTS idx_fills_trade_id ON fills(trade_id);

CREATE OR REPLACE FUNCTION update_strategy_instances_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_strategy_instances_updated_at ON strategy_instances;
CREATE TRIGGER trg_strategy_instances_updated_at
BEFORE UPDATE ON strategy_instances
FOR EACH ROW EXECUTE FUNCTION update_strategy_instances_updated_at();

CREATE OR REPLACE FUNCTION update_research_projects_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_research_projects_updated_at ON research_projects;
CREATE TRIGGER trg_research_projects_updated_at
BEFORE UPDATE ON research_projects
FOR EACH ROW EXECUTE FUNCTION update_research_projects_updated_at();

CREATE OR REPLACE FUNCTION update_gate_status_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_gate_status_updated_at ON gate_status;
CREATE TRIGGER trg_gate_status_updated_at
BEFORE UPDATE ON gate_status
FOR EACH ROW EXECUTE FUNCTION update_gate_status_updated_at();

CREATE OR REPLACE FUNCTION update_order_intents_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_order_intents_updated_at ON order_intents;
CREATE TRIGGER trg_order_intents_updated_at
BEFORE UPDATE ON order_intents
FOR EACH ROW EXECUTE FUNCTION update_order_intents_updated_at();
