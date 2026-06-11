DROP VIEW IF EXISTS robust_event_funnel_summary;
DROP VIEW IF EXISTS strategy_performance_summary;

DROP TABLE IF EXISTS risk_simulation_runs;
DROP TABLE IF EXISTS trade_reviews;
DROP TABLE IF EXISTS walkaway_decisions;
DROP TABLE IF EXISTS strategy_playbook_results;
DROP TABLE IF EXISTS position_size_recommendations;
DROP TABLE IF EXISTS execution_quality_snapshots;
DROP TABLE IF EXISTS event_confounder_links;
DROP TABLE IF EXISTS confounder_events;

ALTER TABLE macro_events
    DROP COLUMN IF EXISTS economic_calendar_event_id;

DROP TABLE IF EXISTS economic_calendar_events;
DROP TABLE IF EXISTS cross_asset_confirmations;
DROP TABLE IF EXISTS market_regime_snapshots;
