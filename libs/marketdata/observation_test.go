package marketdata

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestEconomicObservationRejectsUnsafeMarketInputs(t *testing.T) {
	asOf := time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC)
	base := EconomicObservation{Symbol: "QQQ", Source: "alpaca", ProviderAt: asOf.Add(-time.Minute), LastProviderAt: asOf.Add(-time.Minute), ReceivedAt: asOf.Add(-30 * time.Second), Timeframe: "1m", Bid: 100, Ask: 101, Last: 100.5, Mode: "QUOTE", AsOf: asOf, Provenance: "provider"}
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

func TestQuoteTemporalContractSeparatesSkewAvailabilityAndFreshness(t *testing.T) {
	t0 := time.Date(2026, 9, 29, 14, 0, 0, 0, time.UTC)
	base := EconomicObservation{ProviderAt: t0.Add(77 * time.Millisecond), LastProviderAt: t0.Add(60 * time.Millisecond), ReceivedAt: t0}
	t.Run("77ms-skew-accepted-only-after-all-source-times", func(t *testing.T) {
		if err := base.ValidateQuoteTemporal(t0.Add(100*time.Millisecond), time.Minute, MaxQuoteClockSkew); err != nil {
			t.Fatalf("valid skewed observation rejected: %v", err)
		}
	})
	t.Run("not-yet-available-rejected", func(t *testing.T) {
		err := base.ValidateQuoteTemporal(t0.Add(50*time.Millisecond), time.Minute, MaxQuoteClockSkew)
		if !errors.Is(err, ErrQuoteNotYetAvailable) {
			t.Fatalf("error=%v, want not-yet-available", err)
		}
	})
	t.Run("251ms-skew-rejected", func(t *testing.T) {
		observation := base
		observation.ProviderAt = t0.Add(251 * time.Millisecond)
		err := observation.ValidateQuoteTemporal(t0.Add(time.Second), time.Minute, MaxQuoteClockSkew)
		if !errors.Is(err, ErrProviderClockSkewExceeded) {
			t.Fatalf("error=%v, want provider clock skew exceeded", err)
		}
	})
	t.Run("fresh-quote-stale-trade-rejected", func(t *testing.T) {
		observation := EconomicObservation{ProviderAt: t0.Add(-time.Second), LastProviderAt: t0.Add(-61 * time.Second), ReceivedAt: t0}
		if err := observation.ValidateQuoteTemporal(t0, time.Minute, MaxQuoteClockSkew); !errors.Is(err, ErrQuoteStale) {
			t.Fatalf("error=%v, want stale trade rejection", err)
		}
	})
	t.Run("stale-quote-fresh-trade-rejected", func(t *testing.T) {
		observation := EconomicObservation{ProviderAt: t0.Add(-61 * time.Second), LastProviderAt: t0.Add(-time.Second), ReceivedAt: t0}
		if err := observation.ValidateQuoteTemporal(t0, time.Minute, MaxQuoteClockSkew); !errors.Is(err, ErrQuoteStale) {
			t.Fatalf("error=%v, want stale quote rejection", err)
		}
	})
	t.Run("both-fresh-and-causal-accepted", func(t *testing.T) {
		observation := EconomicObservation{ProviderAt: t0.Add(-20 * time.Second), LastProviderAt: t0.Add(-10 * time.Second), ReceivedAt: t0}
		if err := observation.ValidateQuoteTemporal(t0, time.Minute, MaxQuoteClockSkew); err != nil {
			t.Fatalf("fresh causal quote rejected: %v", err)
		}
	})
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
