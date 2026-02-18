-- Migration 007: Experience & Judgment Layer (EJLayer) — ADR-0012 Phase 2

CREATE TABLE IF NOT EXISTS market_episodes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    episode_type TEXT NOT NULL,
    symbol TEXT NOT NULL,
    strategy_name TEXT NOT NULL,
    artifact_id UUID REFERENCES strategy_artifacts(id) ON DELETE SET NULL,
    episode_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    context JSONB NOT NULL DEFAULT '{}',
    expectations JSONB NOT NULL DEFAULT '{}',
    confidence NUMERIC(5,4) NOT NULL DEFAULT 0,
    uncertainty_budget NUMERIC(5,4) NOT NULL DEFAULT 1,
    context_dominance TEXT NOT NULL DEFAULT 'unclear',
    sequence_position TEXT NOT NULL DEFAULT 'unknown',
    action_taken TEXT NOT NULL DEFAULT 'abstain',
    outcome JSONB,
    surprise_score NUMERIC(5,4),
    hindsight_notes TEXT,
    decay_weight NUMERIC(7,6) NOT NULL DEFAULT 1.0,
    reinforcement_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_market_episodes_symbol ON market_episodes(symbol);
CREATE INDEX IF NOT EXISTS idx_market_episodes_strategy ON market_episodes(strategy_name);
CREATE INDEX IF NOT EXISTS idx_market_episodes_type ON market_episodes(episode_type);
CREATE INDEX IF NOT EXISTS idx_market_episodes_at ON market_episodes(episode_at DESC);
CREATE INDEX IF NOT EXISTS idx_market_episodes_context_dominance ON market_episodes(context_dominance);

CREATE TABLE IF NOT EXISTS negative_patterns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pattern_name TEXT NOT NULL UNIQUE,
    description TEXT,
    reduces_confidence_by NUMERIC(5,4) NOT NULL DEFAULT 0.1,
    match_criteria JSONB NOT NULL DEFAULT '{}',
    reinforcement_count INT NOT NULL DEFAULT 0,
    decay_weight NUMERIC(7,6) NOT NULL DEFAULT 1.0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS episode_pattern_matches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    episode_id UUID NOT NULL REFERENCES market_episodes(id) ON DELETE CASCADE,
    pattern_id UUID NOT NULL REFERENCES negative_patterns(id) ON DELETE CASCADE,
    matched_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE OR REPLACE FUNCTION update_market_episodes_updated_at()
RETURNS TRIGGER AS $$
BEGIN NEW.updated_at = NOW(); RETURN NEW; END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_market_episodes_updated_at ON market_episodes;
CREATE TRIGGER trg_market_episodes_updated_at
    BEFORE UPDATE ON market_episodes
    FOR EACH ROW EXECUTE FUNCTION update_market_episodes_updated_at();

CREATE OR REPLACE FUNCTION update_negative_patterns_updated_at()
RETURNS TRIGGER AS $$
BEGIN NEW.updated_at = NOW(); RETURN NEW; END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_negative_patterns_updated_at ON negative_patterns;
CREATE TRIGGER trg_negative_patterns_updated_at
    BEFORE UPDATE ON negative_patterns
    FOR EACH ROW EXECUTE FUNCTION update_negative_patterns_updated_at();

CREATE OR REPLACE VIEW recent_episodes AS
    SELECT * FROM market_episodes
    WHERE episode_at > NOW() - INTERVAL '30 days'
    ORDER BY episode_at DESC;
