-- Rollback migration 007: EJLayer

DROP VIEW IF EXISTS recent_episodes;

DROP TRIGGER IF EXISTS trg_negative_patterns_updated_at ON negative_patterns;
DROP FUNCTION IF EXISTS update_negative_patterns_updated_at();

DROP TRIGGER IF EXISTS trg_market_episodes_updated_at ON market_episodes;
DROP FUNCTION IF EXISTS update_market_episodes_updated_at();

DROP TABLE IF EXISTS episode_pattern_matches;
DROP TABLE IF EXISTS negative_patterns;
DROP TABLE IF EXISTS market_episodes;
