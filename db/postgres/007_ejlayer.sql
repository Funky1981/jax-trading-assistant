-- 007_ejlayer.sql: Experience & Judgment Layer (EJLayer) schema
-- Phase 2 of ADR-0012 modular-monolith migration

-- market_episodes: one row per decision (trade or abstention)
CREATE TABLE market_episodes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  episode_type TEXT NOT NULL,           -- 'trade' | 'abstention' | 'deferral'
  symbol TEXT NOT NULL,
  strategy_name TEXT NOT NULL,
  artifact_id UUID REFERENCES strategy_artifacts(id) ON DELETE SET NULL,
  episode_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  context JSONB NOT NULL DEFAULT '{}',  -- market context snapshot
  expectations JSONB NOT NULL DEFAULT '{}', -- pre-action expectations
  confidence NUMERIC(5,4) NOT NULL DEFAULT 0,
  uncertainty_budget NUMERIC(5,4) NOT NULL DEFAULT 1,
  context_dominance TEXT NOT NULL DEFAULT 'unclear', -- see enum below
  sequence_position TEXT NOT NULL DEFAULT 'unknown', -- early/mid/late/exhaustion/unknown
  action_taken TEXT NOT NULL DEFAULT 'abstain',      -- 'buy'/'sell'/'abstain'/'defer'
  outcome JSONB,                        -- nullable, filled in post-resolution
  surprise_score NUMERIC(5,4),          -- nullable, filled in post-resolution
  hindsight_notes TEXT,
  decay_weight NUMERIC(7,6) NOT NULL DEFAULT 1.0,
  reinforcement_count INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_market_episodes_symbol ON market_episodes(symbol);
CREATE INDEX idx_market_episodes_strategy ON market_episodes(strategy_name);
CREATE INDEX idx_market_episodes_type ON market_episodes(episode_type);
CREATE INDEX idx_market_episodes_at ON market_episodes(episode_at DESC);
CREATE INDEX idx_market_episodes_context_dominance ON market_episodes(context_dominance);

-- negative_patterns: encoded fragilities that reduce confidence
CREATE TABLE negative_patterns (
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

-- episode_pattern_matches: link episodes to triggered negative patterns
CREATE TABLE episode_pattern_matches (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  episode_id UUID NOT NULL REFERENCES market_episodes(id) ON DELETE CASCADE,
  pattern_id UUID NOT NULL REFERENCES negative_patterns(id) ON DELETE CASCADE,
  matched_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- update trigger for market_episodes
CREATE OR REPLACE FUNCTION update_market_episodes_updated_at()
RETURNS TRIGGER AS $$
BEGIN NEW.updated_at = NOW(); RETURN NEW; END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_market_episodes_updated_at
  BEFORE UPDATE ON market_episodes
  FOR EACH ROW EXECUTE FUNCTION update_market_episodes_updated_at();

-- update trigger for negative_patterns
CREATE OR REPLACE FUNCTION update_negative_patterns_updated_at()
RETURNS TRIGGER AS $$
BEGIN NEW.updated_at = NOW(); RETURN NEW; END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_negative_patterns_updated_at
  BEFORE UPDATE ON negative_patterns
  FOR EACH ROW EXECUTE FUNCTION update_negative_patterns_updated_at();

-- Useful views
CREATE VIEW recent_episodes AS
  SELECT * FROM market_episodes
  WHERE episode_at > NOW() - INTERVAL '30 days'
  ORDER BY episode_at DESC;

CREATE VIEW high_surprise_episodes AS
  SELECT * FROM market_episodes
  WHERE surprise_score > 0.6
  ORDER BY surprise_score DESC, episode_at DESC;
