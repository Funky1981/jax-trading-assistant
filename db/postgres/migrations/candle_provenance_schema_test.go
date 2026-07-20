package migrations

import (
	"os"
	"strings"
	"testing"
)

func TestCandleProvenanceMigrationPreservesGenuineIdentity(t *testing.T) {
	data, err := os.ReadFile("000048_candle_provenance.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(data)
	for _, required := range []string{
		"timeframe TEXT NOT NULL", "source TEXT NOT NULL", "timestamp_semantics TEXT NOT NULL",
		"regular_trading_hours BOOLEAN", "market_data_classification TEXT NOT NULL",
		"NUMERIC(18,6)", "UPPER(source) NOT IN ('TEST', 'SYNTHETIC', 'FIXTURE')",
		"ON candles(symbol, timeframe, source, timestamp)",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("migration missing %q", required)
		}
	}
}
