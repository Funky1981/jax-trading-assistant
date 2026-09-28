package marketdata

import (
	"context"
	"testing"
	"time"
)

func TestEconomicObservationRejectsUnsafeMarketInputs(t *testing.T) {
	asOf := time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC)
	base := EconomicObservation{Symbol: "QQQ", Source: "alpaca", ProviderAt: asOf.Add(-time.Minute), ReceivedAt: asOf.Add(-30 * time.Second), Timeframe: "1m", Bid: 100, Ask: 101, Last: 100.5, Mode: "QUOTE", AsOf: asOf, Provenance: "provider"}
	validate := func(observation EconomicObservation, timeframe string, allowed []string) error {
		return observation.ValidateFor("QQQ", timeframe, asOf, 2*time.Minute, allowed, true)
	}
	if err := validate(base, "1m", []string{"alpaca"}); err != nil {
		t.Fatalf("valid observation rejected: %v", err)
	}
	cases := []struct {
		name    string
		edit    func(*EconomicObservation)
		frame   string
		allowed []string
	}{
		{"future", func(o *EconomicObservation) {
			o.ProviderAt = asOf.Add(time.Second)
			o.ReceivedAt = asOf.Add(time.Second)
		}, "1m", []string{"alpaca"}},
		{"stale", func(o *EconomicObservation) { o.ProviderAt = asOf.Add(-3 * time.Minute) }, "1m", []string{"alpaca"}},
		{"unknown source", func(o *EconomicObservation) { o.Source = "unknown" }, "1m", []string{"unknown"}},
		{"fixture source", func(o *EconomicObservation) { o.Source = "fixture-alpaca" }, "1m", []string{"fixture-alpaca"}},
		{"wrong symbol", func(o *EconomicObservation) { o.Symbol = "SPY" }, "1m", []string{"alpaca"}},
		{"wrong timeframe", func(o *EconomicObservation) {}, "1d", []string{"alpaca"}},
		{"source not allowed", func(o *EconomicObservation) {}, "1m", []string{"polygon"}},
		{"bad price", func(o *EconomicObservation) { o.Bid = 0 }, "1m", []string{"alpaca"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			observation := base
			tc.edit(&observation)
			if err := validate(observation, tc.frame, tc.allowed); err == nil {
				t.Fatal("unsafe observation accepted")
			}
		})
	}
}

type quoteSourceProbe struct{}

func (quoteSourceProbe) Name() string { return "AlPaCa" }
func (quoteSourceProbe) GetQuote(_ context.Context, symbol string) (*Quote, error) {
	return &Quote{Symbol: symbol, Price: 100, Timestamp: time.Now().UTC()}, nil
}
func (quoteSourceProbe) GetCandles(context.Context, string, Timeframe, int) ([]Candle, error) {
	return nil, nil
}
func (quoteSourceProbe) GetTrades(context.Context, string, int) ([]Trade, error) { return nil, nil }
func (quoteSourceProbe) GetEarnings(context.Context, string, int) ([]Earnings, error) {
	return nil, nil
}
func (quoteSourceProbe) StreamQuotes(context.Context, []string) (<-chan StreamUpdate, error) {
	return nil, nil
}
func (quoteSourceProbe) HealthCheck(context.Context) error { return nil }

func TestGetQuoteWithSourceReturnsActualProviderIdentity(t *testing.T) {
	client := &Client{providers: []Provider{quoteSourceProbe{}}}
	quote, source, err := client.GetQuoteWithSource(context.Background(), "QQQ")
	if err != nil || quote == nil || source != "alpaca" {
		t.Fatalf("quote=%+v source=%q err=%v", quote, source, err)
	}
}
