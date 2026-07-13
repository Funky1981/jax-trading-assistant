CREATE TABLE IF NOT EXISTS candidate_paper_tickets (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	paper_ticket_id TEXT NOT NULL UNIQUE,
	candidate_id UUID NOT NULL REFERENCES candidate_trades(id) ON DELETE CASCADE,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	status TEXT NOT NULL DEFAULT 'paper_ticket_created',
	source_approval_id UUID REFERENCES candidate_approvals(id) ON DELETE SET NULL,
	approval_decision_ref TEXT,
	symbol TEXT NOT NULL,
	direction TEXT NOT NULL,
	setup_type TEXT NOT NULL,
	catalyst_summary TEXT NOT NULL,
	entry_price NUMERIC(18,6) NOT NULL,
	stop_loss_price NUMERIC(18,6) NOT NULL,
	target_price NUMERIC(18,6) NOT NULL,
	position_size NUMERIC(18,6) NOT NULL,
	max_normal_loss NUMERIC(18,6) NOT NULL,
	max_slippage_adjusted_loss NUMERIC(18,6) NOT NULL,
	reward_risk_ratio NUMERIC(18,6) NOT NULL,
	evidence_status TEXT NOT NULL,
	gate_status TEXT NOT NULL,
	risk_status TEXT NOT NULL,
	approval_status TEXT NOT NULL,
	paper_only BOOLEAN NOT NULL DEFAULT TRUE,
	broker_execution_allowed BOOLEAN NOT NULL DEFAULT FALSE,
	execution_instruction_created BOOLEAN NOT NULL DEFAULT FALSE,
	live_trading_allowed BOOLEAN NOT NULL DEFAULT FALSE,
	leverage_allowed BOOLEAN NOT NULL DEFAULT FALSE,
	reject_reasons TEXT[] NOT NULL DEFAULT '{}',
	warning_reasons TEXT[] NOT NULL DEFAULT '{}',
	CONSTRAINT uq_candidate_paper_tickets_candidate UNIQUE (candidate_id),
	CONSTRAINT chk_candidate_paper_tickets_status CHECK (
		status IN ('paper_ticket_ready', 'paper_ticket_created', 'paper_ticket_reviewed', 'paper_ticket_cancelled', 'paper_ticket_blocked')
	),
	CONSTRAINT chk_candidate_paper_tickets_paper_only CHECK (paper_only = TRUE),
	CONSTRAINT chk_candidate_paper_tickets_no_broker CHECK (broker_execution_allowed = FALSE),
	CONSTRAINT chk_candidate_paper_tickets_no_execution CHECK (execution_instruction_created = FALSE),
	CONSTRAINT chk_candidate_paper_tickets_no_live CHECK (live_trading_allowed = FALSE),
	CONSTRAINT chk_candidate_paper_tickets_no_leverage CHECK (leverage_allowed = FALSE)
);

CREATE INDEX IF NOT EXISTS idx_candidate_paper_tickets_candidate
	ON candidate_paper_tickets(candidate_id);

CREATE INDEX IF NOT EXISTS idx_candidate_paper_tickets_status_created
	ON candidate_paper_tickets(status, created_at DESC);
