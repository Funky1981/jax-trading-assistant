package papertrading

import (
	"fmt"
	"time"
)

type MarketTick struct {
	TickID            string    `json:"tick_id"`
	InstrumentID      string    `json:"instrument_id"`
	Bid               float64   `json:"bid"`
	Ask               float64   `json:"ask"`
	Last              float64   `json:"last"`
	AvailableQuantity float64   `json:"available_quantity"`
	Timestamp         time.Time `json:"timestamp"`
	ReceivedAt        time.Time `json:"received_at"`
	Session           Session   `json:"session"`
	Source            string    `json:"source"`
}

func (tick MarketTick) Validate(maxAge time.Duration) error {
	if tick.TickID == "" || tick.InstrumentID == "" || tick.Source == "" || tick.Session == SessionUnknown {
		return fmt.Errorf("%w: tick identity or session is unknown", ErrInvalidMarketData)
	}
	if tick.Session != SessionOpen && tick.Session != SessionClosed {
		return fmt.Errorf("%w: unsupported session", ErrInvalidMarketData)
	}
	if !finitePositive(tick.Bid) || !finitePositive(tick.Ask) || !finitePositive(tick.Last) || tick.Bid > tick.Ask || !finiteNonNegative(tick.AvailableQuantity) {
		return fmt.Errorf("%w: prices or available quantity are invalid", ErrInvalidMarketData)
	}
	if tick.Timestamp.IsZero() || tick.ReceivedAt.IsZero() || tick.Timestamp.Location() != time.UTC || tick.ReceivedAt.Location() != time.UTC {
		return fmt.Errorf("%w: timestamps must be UTC", ErrInvalidMarketData)
	}
	if tick.ReceivedAt.Before(tick.Timestamp) {
		return fmt.Errorf("%w: received before market timestamp", ErrInvalidMarketData)
	}
	if maxAge <= 0 || tick.ReceivedAt.Sub(tick.Timestamp) > maxAge {
		return ErrStaleMarketData
	}
	return nil
}
