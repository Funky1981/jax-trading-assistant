-- Rollback Codex Packs additive schema (000008)

DROP TRIGGER IF EXISTS trg_order_intents_updated_at ON order_intents;
DROP FUNCTION IF EXISTS update_order_intents_updated_at();

DROP TRIGGER IF EXISTS trg_gate_status_updated_at ON gate_status;
DROP FUNCTION IF EXISTS update_gate_status_updated_at();

DROP TRIGGER IF EXISTS trg_research_projects_updated_at ON research_projects;
DROP FUNCTION IF EXISTS update_research_projects_updated_at();

DROP TRIGGER IF EXISTS trg_strategy_instances_updated_at ON strategy_instances;
DROP FUNCTION IF EXISTS update_strategy_instances_updated_at();

DROP TABLE IF EXISTS fills;
DROP TABLE IF EXISTS order_intents;
DROP TABLE IF EXISTS gate_status;
DROP TABLE IF EXISTS test_runs;
DROP TABLE IF EXISTS ai_decision_acceptance;
DROP TABLE IF EXISTS ai_decisions;
DROP TABLE IF EXISTS run_artifacts;
DROP TABLE IF EXISTS runs;
DROP TABLE IF EXISTS research_project_runs;
DROP TABLE IF EXISTS research_projects;
DROP TABLE IF EXISTS backtest_trades;
DROP TABLE IF EXISTS backtest_runs;

ALTER TABLE IF EXISTS strategy_signals
    DROP CONSTRAINT IF EXISTS fk_strategy_signals_instance;
ALTER TABLE IF EXISTS trades
    DROP CONSTRAINT IF EXISTS fk_trades_instance;

DROP INDEX IF EXISTS idx_strategy_signals_instance_generated_at;
DROP INDEX IF EXISTS idx_trades_instance_created_at;

ALTER TABLE IF EXISTS strategy_signals
    DROP COLUMN IF EXISTS instance_id;
ALTER TABLE IF EXISTS trades
    DROP COLUMN IF EXISTS instance_id;

DELETE FROM strategy_instances
WHERE id = '00000000-0000-0000-0000-000000000001'::uuid;

DROP TABLE IF EXISTS strategy_instances;
