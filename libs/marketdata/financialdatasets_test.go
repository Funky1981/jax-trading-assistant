package marketdata

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFinancialDatasetsDateRangeIncludesRequiredDates(t *testing.T) {
	now := time.Date(2026, 3, 6, 12, 0, 0, 0, time.UTC)
	startDate, endDate := financialDatasetsDateRange(Timeframe15Min, 100, now)

	if startDate == "" || endDate == "" {
		t.Fatal("expected start_date and end_date to be populated")
	}
	if endDate != "2026-03-06" {
		t.Fatalf("expected end_date 2026-03-06, got %s", endDate)
	}
}

func TestFinancialDatasetsGetCandlesSendsStartAndEndDate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("ticker") != "AAPL" {
			t.Fatalf("expected ticker AAPL, got %s", query.Get("ticker"))
		}
		if query.Get("start_date") == "" {
			t.Fatal("expected start_date query param")
		}
		if query.Get("end_date") == "" {
			t.Fatal("expected end_date query param")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"prices":[{"time":"2026-03-05T15:30:00Z","open":1,"high":2,"low":0.5,"close":1.5,"volume":100}]}`))
	}))
	defer server.Close()

	provider, err := NewFinancialDatasetsProvider(ProviderConfig{
		Name:    ProviderFinancialDatasets,
		APIKey:  "test-key",
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("unexpected provider init error: %v", err)
	}
	provider.baseURL = server.URL
	provider.client = server.Client()

	candles, err := provider.GetCandles(context.Background(), "AAPL", Timeframe15Min, 100)
	if err != nil {
		t.Fatalf("unexpected GetCandles error: %v", err)
	}
	if len(candles) != 1 {
		t.Fatalf("expected 1 candle, got %d", len(candles))
	}
	if candles[0].Close != 1.5 {
		t.Fatalf("expected close 1.5, got %v", candles[0].Close)
	}
}
