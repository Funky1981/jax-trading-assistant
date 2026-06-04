package migrations_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSentimentFeatureSchema(t *testing.T) {
	path := filepath.Join("000029_sentiment_feature_layer.up.sql")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	sql := string(data)
	for _, required := range []string{
		"CREATE TABLE IF NOT EXISTS news_items",
		"source_family TEXT NOT NULL",
		"symbol TEXT NOT NULL",
		"CREATE TABLE IF NOT EXISTS sentiment_events",
		"provider_mode TEXT NOT NULL",
		"score NUMERIC NOT NULL",
		"confidence NUMERIC NOT NULL",
		"drivers JSONB NOT NULL DEFAULT '[]'::jsonb",
		"limitations JSONB NOT NULL DEFAULT '[]'::jsonb",
		"CREATE TABLE IF NOT EXISTS sentiment_aggregates",
		"time_window TEXT NOT NULL",
		"source_groups JSONB NOT NULL DEFAULT '{}'::jsonb",
		"state TEXT NOT NULL DEFAULT 'available'",
		"CREATE TABLE IF NOT EXISTS opportunity_sentiment_snapshots",
		"snapshot_reason TEXT NOT NULL DEFAULT 'created'",
		"CREATE TABLE IF NOT EXISTS sentiment_alert_rules",
		"cooldown_seconds INT NOT NULL DEFAULT 86400",
		"CREATE TABLE IF NOT EXISTS notification_events",
		"sentiment_trigger_type TEXT",
		"CREATE TABLE IF NOT EXISTS notification_preferences",
		"CREATE TABLE IF NOT EXISTS approval_override_reasons",
		"sentiment_evidence_viewed BOOLEAN NOT NULL DEFAULT FALSE",
		"CREATE TABLE IF NOT EXISTS sentiment_paper_live_handoffs",
		"live_ready BOOLEAN NOT NULL DEFAULT FALSE",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("sentiment schema missing %q", required)
		}
	}
}
