CREATE TABLE IF NOT EXISTS candidate_evidence_items (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	evidence_id UUID NOT NULL UNIQUE,
	candidate_id UUID NOT NULL REFERENCES candidate_trades(id) ON DELETE CASCADE,
	source_type TEXT NOT NULL,
	source_ref TEXT NOT NULL,
	observed_at TIMESTAMPTZ NOT NULL,
	summary TEXT NOT NULL,
	evidence_kind TEXT NOT NULL,
	supports_candidate BOOLEAN NOT NULL DEFAULT FALSE,
	contradicts_candidate BOOLEAN NOT NULL DEFAULT FALSE,
	confidence NUMERIC(8,6) NOT NULL,
	impact_score NUMERIC(8,6) NOT NULL,
	quality_score NUMERIC(8,6) NOT NULL,
	freshness_status TEXT NOT NULL,
	notes TEXT,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	CONSTRAINT chk_candidate_evidence_direction CHECK (
		supports_candidate OR contradicts_candidate
	),
	CONSTRAINT chk_candidate_evidence_freshness CHECK (
		freshness_status IN ('fresh', 'stale', 'critical_stale')
	)
);

CREATE INDEX IF NOT EXISTS idx_candidate_evidence_items_candidate
	ON candidate_evidence_items(candidate_id, observed_at DESC);

CREATE TABLE IF NOT EXISTS candidate_evidence_scores (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	candidate_id UUID NOT NULL REFERENCES candidate_trades(id) ON DELETE CASCADE,
	support_score NUMERIC(8,6) NOT NULL,
	contradiction_score NUMERIC(8,6) NOT NULL,
	quality_score NUMERIC(8,6) NOT NULL,
	freshness_score NUMERIC(8,6) NOT NULL,
	overall_evidence_score NUMERIC(8,6) NOT NULL,
	evidence_item_count INTEGER NOT NULL,
	supporting_item_count INTEGER NOT NULL,
	contradictory_item_count INTEGER NOT NULL,
	stale_item_count INTEGER NOT NULL,
	evidence_status TEXT NOT NULL,
	evidence_ready BOOLEAN NOT NULL DEFAULT FALSE,
	evidence_gate_ready BOOLEAN NOT NULL DEFAULT FALSE,
	broker_execution_allowed BOOLEAN NOT NULL DEFAULT FALSE,
	execution_instruction_created BOOLEAN NOT NULL DEFAULT FALSE,
	approval_granted BOOLEAN NOT NULL DEFAULT FALSE,
	scored_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	CONSTRAINT chk_candidate_evidence_status CHECK (
		evidence_status IN ('missing', 'weak', 'mixed', 'sufficient', 'stale', 'blocked')
	)
);

CREATE INDEX IF NOT EXISTS idx_candidate_evidence_scores_candidate
	ON candidate_evidence_scores(candidate_id, scored_at DESC);
