package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"jax-trading-assistant/libs/marketdata"
)

func TestGenuineProviderAcceptance(t *testing.T) {
	for _, source := range []string{"ib-bridge", "alpaca", "polygon"} {
		if !genuineProvider(source) {
			t.Fatalf("genuine source %q rejected", source)
		}
	}
	for _, source := range []string{"", "unknown", "TEST", "SYNTHETIC", "FIXTURE"} {
		if genuineProvider(strings.ToLower(source)) {
			t.Fatalf("unsafe source %q accepted", source)
		}
	}
}

func TestPrepareGenuineCandlesPreservesTimestampSymbolAndPrecision(t *testing.T) {
	from := time.Date(2026, 7, 17, 16, 46, 31, 0, time.UTC)
	at := from.Add(time.Hour)
	input := []marketdata.Candle{{Symbol: "qqq", Timestamp: at, Open: 500.123456, High: 501.234567, Low: 499.123456, Close: 500.654321, Volume: 10}}
	got := prepareGenuineCandles("QQQ", from, at.Add(time.Hour), input)
	if len(got) != 1 {
		t.Fatalf("count=%d", len(got))
	}
	if got[0].Symbol != "QQQ" {
		t.Fatalf("symbol=%q", got[0].Symbol)
	}
	if !got[0].Timestamp.Equal(at) {
		t.Fatalf("timestamp=%s", got[0].Timestamp)
	}
	if got[0].Close != 500.654321 {
		t.Fatalf("close=%v", got[0].Close)
	}
}

func TestPrepareGenuineCandlesDoesNotFabricateOrDuplicateIntervals(t *testing.T) {
	from := time.Date(2026, 7, 17, 16, 46, 31, 0, time.UTC)
	at := from.Add(time.Hour)
	got := prepareGenuineCandles("QQQ", from, at.Add(2*time.Hour), []marketdata.Candle{{Timestamp: at, Open: 1, High: 2, Low: 1, Close: 2}, {Timestamp: at, Open: 1, High: 2, Low: 1, Close: 2}})
	if len(got) != 1 {
		t.Fatalf("count=%d, want only supplied unique observation", len(got))
	}
}

func TestPrepareGenuineCandlesDoesNotInventMarketClosedPeriods(t *testing.T) {
	friday := time.Date(2026, 7, 17, 19, 0, 0, 0, time.UTC)
	monday := time.Date(2026, 7, 20, 13, 0, 0, 0, time.UTC)
	got := prepareGenuineCandles("QQQ", friday, monday.Add(time.Hour), []marketdata.Candle{{Timestamp: friday, Open: 1, High: 2, Low: 1, Close: 2}, {Timestamp: monday, Open: 2, High: 3, Low: 2, Close: 3}})
	if len(got) != 2 {
		t.Fatalf("count=%d; weekend intervals must not be invented", len(got))
	}
}

func TestGenuineCandleCollectionHasNoExecutionOrRuntimeMutation(t *testing.T) {
	var source string
	for _, path := range []string{"genuine_candle_collection.go", "evaluation_market_coverage.go"} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		source += strings.ToLower(string(data))
	}
	for _, forbidden := range []string{"insert into execution_instructions", "insert into order_intents", "insert into trades", "insert into fills", "broker_order_id", "allow_live_trading=true", "execution_instruction_worker_enabled=true"} {
		if strings.Contains(source, forbidden) {
			t.Fatalf("collection path contains forbidden mutation %q", forbidden)
		}
	}
}

func TestEvaluationCoverageRejectsUnboundedSymbolSetBeforeDatabaseAccess(t *testing.T) {
	handler := evaluationMarketCoverageHandler(nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/market/candles/collect-evaluation", bytes.NewBufferString(`{"maxSymbols":26}`))
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want %d", res.Code, http.StatusBadRequest)
	}
}

func TestProviderCandleProvenanceDoesNotInventAdjustmentState(t *testing.T) {
	adjusted, timezone := providerCandleProvenance("alpaca")
	if adjusted != "unknown" || timezone != "UTC" {
		t.Fatalf("adjusted=%q timezone=%q", adjusted, timezone)
	}
}

func TestEvaluationCoverageHasBoundedRetryAndTimeout(t *testing.T) {
	data, err := os.ReadFile("evaluation_market_coverage.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	for _, required := range []string{"attempt <= 2", "context.WithTimeout", "45*time.Second"} {
		if !strings.Contains(source, required) {
			t.Fatalf("bounded coverage missing %q", required)
		}
	}
}

func TestDefaultMarketSymbolsIncludeQQQ(t *testing.T) {
	t.Setenv("MARKET_SYMBOLS", "")
	cfg := loadIngesterConfig()
	found := false
	for _, s := range cfg.Symbols {
		if s == "QQQ" {
			found = true
		}
	}
	if !found {
		t.Fatal("QQQ missing from default market symbols")
	}
}
