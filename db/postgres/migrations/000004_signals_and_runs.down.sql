-- Rollback strategy signals, orchestration runs, and trade approvals tables

DROP TABLE IF EXISTS trade_approvals CASCADE;
DROP TABLE IF EXISTS strategy_signals CASCADE;
DROP TABLE IF EXISTS orchestration_runs CASCADE;
