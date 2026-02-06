-- Add tables for strategy signals, orchestration runs, and trade approvals

-- Strategy signals (trading opportunities detected by strategies)
CREATE TABLE IF NOT EXISTS strategy_signals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    symbol VARCHAR(10) NOT NULL,
    strategy_id VARCHAR(50) NOT NULL,
    signal_type VARCHAR(10) NOT NULL CHECK (signal_type IN ('BUY', 'SELL', 'HOLD')),
    confidence DECIMAL(3,2) NOT NULL CHECK (confidence >= 0.00 AND confidence <= 1.00),
    entry_price DECIMAL(12,2),
    stop_loss DECIMAL(12,2),
    take_profit DECIMAL(12,2),
    reasoning TEXT,
    generated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ,
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected', 'expired', 'cancelled')),
    orchestration_run_id UUID,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_signals_status ON strategy_signals(status);
CREATE INDEX IF NOT EXISTS idx_signals_symbol ON strategy_signals(symbol);
CREATE INDEX IF NOT EXISTS idx_signals_generated_at ON strategy_signals(generated_at DESC);
CREATE INDEX IF NOT EXISTS idx_signals_expires_at ON strategy_signals(expires_at);
CREATE INDEX IF NOT EXISTS idx_signals_strategy ON strategy_signals(strategy_id);

-- Orchestration runs (AI analysis tracking)
CREATE TABLE IF NOT EXISTS orchestration_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    symbol VARCHAR(10) NOT NULL,
    trigger_type VARCHAR(50), -- 'signal', 'scheduled', 'manual'
    trigger_id UUID,
    agent_suggestion TEXT,
    confidence DECIMAL(3,2) CHECK (confidence >= 0.00 AND confidence <= 1.00),
    reasoning TEXT,
    memories_recalled INT DEFAULT 0,
    status VARCHAR(20) DEFAULT 'running' CHECK (status IN ('running', 'completed', 'failed', 'cancelled')),
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ,
    error TEXT,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_orchestration_runs_status ON orchestration_runs(status);
CREATE INDEX IF NOT EXISTS idx_orchestration_runs_symbol ON orchestration_runs(symbol);
CREATE INDEX IF NOT EXISTS idx_orchestration_runs_started_at ON orchestration_runs(started_at DESC);
CREATE INDEX IF NOT EXISTS idx_orchestration_runs_trigger ON orchestration_runs(trigger_type, trigger_id);

-- Trade approvals (user decisions on signals)
CREATE TABLE IF NOT EXISTS trade_approvals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    signal_id UUID REFERENCES strategy_signals(id) ON DELETE CASCADE,
    orchestration_run_id UUID REFERENCES orchestration_runs(id) ON DELETE SET NULL,
    approved BOOLEAN NOT NULL,
    approved_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    approved_by VARCHAR(100),
    modification_notes TEXT,
    order_id VARCHAR(100), -- IB order ID if approved and executed
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_trade_approvals_signal ON trade_approvals(signal_id);
CREATE INDEX IF NOT EXISTS idx_trade_approvals_approved_at ON trade_approvals(approved_at DESC);
CREATE INDEX IF NOT EXISTS idx_trade_approvals_approved ON trade_approvals(approved);

-- Foreign key for orchestration_run_id in strategy_signals (added after table creation to avoid circular dependency)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_signals_orchestration_run') THEN
        ALTER TABLE strategy_signals 
        ADD CONSTRAINT fk_signals_orchestration_run 
        FOREIGN KEY (orchestration_run_id) 
        REFERENCES orchestration_runs(id) 
        ON DELETE SET NULL;
    END IF;
END $$;
