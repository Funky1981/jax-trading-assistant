package migrations

import (
	"os"
	"strings"
	"testing"
)

func TestRobustProfitabilityLayerSchema(t *testing.T) {
	up, err := os.ReadFile("000042_robust_profitability_layer.up.sql")
	if err != nil {
		t.Fatalf("read up migration: %v", err)
	}
	sql := string(up)

	for _, table := range []string{
		"market_regime_snapshots",
		"cross_asset_confirmations",
		"economic_calendar_events",
		"confounder_events",
		"event_confounder_links",
		"execution_quality_snapshots",
		"position_size_recommendations",
		"strategy_playbook_results",
		"walkaway_decisions",
		"trade_reviews",
		"risk_simulation_runs",
	} {
		requireRobustSchemaContains(t, sql, "CREATE TABLE IF NOT EXISTS "+table)
	}

	for _, fragment := range []string{
		"ALTER TABLE macro_events",
		"economic_calendar_event_id UUID NULL REFERENCES economic_calendar_events(id)",
		"CREATE OR REPLACE VIEW strategy_performance_summary",
		"CREATE OR REPLACE VIEW robust_event_funnel_summary",
		"chk_market_regime_snapshots_primary",
		"chk_cross_asset_confirmations_verdict",
		"chk_execution_quality_snapshots_verdict",
		"chk_position_size_recommendations_verdict",
		"chk_strategy_playbook_results_result",
		"chk_walkaway_decisions_severity",
		"chk_trade_reviews_outcome",
	} {
		requireRobustSchemaContains(t, sql, fragment)
	}
}

func TestRobustProfitabilityLayerDownMigration(t *testing.T) {
	down, err := os.ReadFile("000042_robust_profitability_layer.down.sql")
	if err != nil {
		t.Fatalf("read down migration: %v", err)
	}
	sql := string(down)

	for _, fragment := range []string{
		"DROP VIEW IF EXISTS robust_event_funnel_summary",
		"DROP VIEW IF EXISTS strategy_performance_summary",
		"DROP TABLE IF EXISTS risk_simulation_runs",
		"DROP COLUMN IF EXISTS economic_calendar_event_id",
		"DROP TABLE IF EXISTS market_regime_snapshots",
	} {
		requireRobustSchemaContains(t, sql, fragment)
	}
}

func requireRobustSchemaContains(t *testing.T, sql string, fragment string) {
	t.Helper()
	if !strings.Contains(sql, fragment) {
		t.Fatalf("schema missing %q", fragment)
	}
}
