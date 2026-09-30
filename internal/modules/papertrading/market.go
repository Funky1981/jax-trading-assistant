package papertrading

import (
	"errors"
	"fmt"
	"time"

	"jax-trading-assistant/libs/marketdata"
)

type MarketTick struct {
	TickID                    string    `json:"tick_id"`
	InstrumentID              string    `json:"instrument_id"`
	MarketSymbol              string    `json:"market_symbol,omitempty"`
	Bid                       float64   `json:"bid"`
	Ask                       float64   `json:"ask"`
	Last                      float64   `json:"last"`
	AvailableQuantity         float64   `json:"available_quantity"`
	Timestamp                 time.Time `json:"timestamp"`
	LastProviderAt            time.Time `json:"last_provider_at,omitempty"`
	ReceivedAt                time.Time `json:"received_at"`
	AsOf                      time.Time `json:"as_of,omitempty"`
	RequireTemporalProvenance bool      `json:"require_temporal_provenance,omitempty"`
	Session                   Session   `json:"session"`
	Source                    string    `json:"source"`
}

// MarketProvenance records the immutable source times used by a canonical
// quote-based execution. AvailableAt is the first instant when every source
// timestamp and local receipt is causally observable.
type MarketProvenance struct {
	Provider        string    `json:"provider"`
	MarketSymbol    string    `json:"market_symbol,omitempty"`
	QuoteProviderAt time.Time `json:"quote_provider_at"`
	TradeProviderAt time.Time `json:"trade_provider_at"`
	ReceivedAt      time.Time `json:"received_at"`
	AsOf            time.Time `json:"as_of"`
	AvailableAt     time.Time `json:"available_at"`
	StrictTemporal  bool      `json:"strict_temporal"`
}

func (tick MarketTick) Provenance() (MarketProvenance, error) {
	availableAt, err := tick.AvailableAt()
	if err != nil {
		return MarketProvenance{}, err
	}
	if !tick.RequireTemporalProvenance || tick.LastProviderAt.IsZero() || tick.AsOf.IsZero() {
		return MarketProvenance{}, fmt.Errorf("%w: strict quote/trade/receipt/as-of provenance is required", ErrInvalidMarketData)
	}
	return MarketProvenance{
		Provider: tick.Source, MarketSymbol: tick.MarketSymbol, QuoteProviderAt: tick.Timestamp.UTC(),
		TradeProviderAt: tick.LastProviderAt.UTC(), ReceivedAt: tick.ReceivedAt.UTC(),
		AsOf: tick.AsOf.UTC(), AvailableAt: availableAt.UTC(), StrictTemporal: true,
	}, nil
}

func (tick MarketTick) Validate(maxAge time.Duration) error {
	if tick.TickID == "" || tick.InstrumentID == "" || tick.Source == "" {
		return fmt.Errorf("%w: tick identity is unknown", ErrInvalidMarketData)
	}
	if tick.Session != SessionOpen && tick.Session != SessionClosed && tick.Session != SessionUnknown {
		return fmt.Errorf("%w: unsupported session", ErrInvalidMarketData)
	}
	if !finitePositive(tick.Bid) || !finitePositive(tick.Ask) || !finitePositive(tick.Last) || tick.Bid > tick.Ask || !finiteNonNegative(tick.AvailableQuantity) {
		return fmt.Errorf("%w: prices or available quantity are invalid", ErrInvalidMarketData)
	}
	if tick.Timestamp.IsZero() || tick.ReceivedAt.IsZero() || tick.Timestamp.Location() != time.UTC || tick.ReceivedAt.Location() != time.UTC {
		return fmt.Errorf("%w: timestamps must be UTC", ErrInvalidMarketData)
	}
	lastProviderAt := tick.LastProviderAt
	asOf := tick.AsOf
	if tick.RequireTemporalProvenance && (lastProviderAt.IsZero() || asOf.IsZero() || lastProviderAt.Location() != time.UTC || asOf.Location() != time.UTC) {
		return fmt.Errorf("%w: canonical quote, trade, receipt, and as-of timestamps are required", ErrInvalidMarketData)
	}
	// Legacy synthetic/candle ticks retain their historical compatibility
	// defaults. Genuine canonical quote ticks opt into strict provenance above.
	if lastProviderAt.IsZero() {
		lastProviderAt = tick.Timestamp
	}
	if asOf.IsZero() {
		asOf = tick.ReceivedAt
	}
	if err := marketdata.ValidateQuoteTemporal(tick.Timestamp, lastProviderAt, tick.ReceivedAt, asOf, maxAge, marketdata.MaxQuoteClockSkew); err != nil {
		if errors.Is(err, marketdata.ErrQuoteStale) {
			return ErrStaleMarketData
		}
		return fmt.Errorf("%w: %v", ErrInvalidMarketData, err)
	}
	return nil
}

// AvailableAt returns the first instant when all source and local
// receipt timestamps for this observation are available. Source timestamps
// remain unchanged; compatibility ticks without a trade timestamp use their
// quote timestamp only when strict provenance was not requested.
func (tick MarketTick) AvailableAt() (time.Time, error) {
	if tick.Timestamp.IsZero() || tick.ReceivedAt.IsZero() || (tick.RequireTemporalProvenance && (tick.LastProviderAt.IsZero() || tick.AsOf.IsZero())) {
		return time.Time{}, fmt.Errorf("%w: causal market timestamps are incomplete", ErrInvalidMarketData)
	}
	availableAt := tick.Timestamp
	if tick.LastProviderAt.IsZero() {
		if tick.RequireTemporalProvenance {
			return time.Time{}, fmt.Errorf("%w: latest-trade timestamp is required", ErrInvalidMarketData)
		}
	} else if tick.LastProviderAt.After(availableAt) {
		availableAt = tick.LastProviderAt
	}
	if tick.ReceivedAt.After(availableAt) {
		availableAt = tick.ReceivedAt
	}
	return availableAt.UTC(), nil
}
