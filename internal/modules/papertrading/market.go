package papertrading

import (
	"errors"
	"fmt"
	"time"

	"jax-trading-assistant/libs/marketdata"
)

type MarketTick struct {
	TickID            string    `json:"tick_id"`
	InstrumentID      string    `json:"instrument_id"`
	Bid               float64   `json:"bid"`
	Ask               float64   `json:"ask"`
	Last              float64   `json:"last"`
	AvailableQuantity float64   `json:"available_quantity"`
	Timestamp         time.Time `json:"timestamp"`
	LastProviderAt    time.Time `json:"last_provider_at,omitempty"`
	ReceivedAt        time.Time `json:"received_at"`
	AsOf              time.Time `json:"as_of,omitempty"`
	Session           Session   `json:"session"`
	Source            string    `json:"source"`
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
	if lastProviderAt.IsZero() {
		lastProviderAt = tick.Timestamp
	}
	asOf := tick.AsOf
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
