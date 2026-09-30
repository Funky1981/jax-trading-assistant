package marketdata

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

const MaxQuoteClockSkew = 250 * time.Millisecond

var ErrProviderClockSkewExceeded = errors.New("PROVIDER_CLOCK_SKEW_EXCEEDED")
var ErrQuoteNotYetAvailable = errors.New("market observation is not yet available at asOf")
var ErrQuoteStale = errors.New("quote or trade observation is stale")

// EconomicObservation is the bounded market input used by economic decisions.
// It carries provider and acquisition time separately from exchange identity.
type EconomicObservation struct {
	Symbol         string        `json:"symbol"`
	Source         string        `json:"source"`
	ProviderAt     time.Time     `json:"providerAt"`
	LastProviderAt time.Time     `json:"lastProviderAt,omitempty"`
	ReceivedAt     time.Time     `json:"receivedAt"`
	CompletedAt    time.Time     `json:"completedAt,omitempty"`
	Timeframe      string        `json:"timeframe,omitempty"`
	Bid            float64       `json:"bid,omitempty"`
	Ask            float64       `json:"ask,omitempty"`
	BidSize        float64       `json:"bidSize,omitempty"`
	AskSize        float64       `json:"askSize,omitempty"`
	Last           float64       `json:"last"`
	Mode           string        `json:"mode"`
	AsOf           time.Time     `json:"asOf"`
	FreshnessAge   time.Duration `json:"freshnessAge"`
	Provenance     string        `json:"provenance"`
}

// ValidateFor enforces source, identity, temporal, freshness and price
// constraints. AllowedSources must be an explicit non-empty allowlist.
func (observation EconomicObservation) ValidateFor(symbol, timeframe string, asOf time.Time, maxAge time.Duration, allowedSources []string, requireQuoteSides bool) error {
	return observation.ValidateForMode(symbol, timeframe, asOf, maxAge, allowedSources, requireQuoteSides, false)
}

// ValidateForMode has an explicit non-production escape hatch for tests that
// inject a disposable fixture policy. Production policy loading never enables it.
func (observation EconomicObservation) ValidateForMode(symbol, timeframe string, asOf time.Time, maxAge time.Duration, allowedSources []string, requireQuoteSides, allowNonProductionSources bool) error {
	if len(allowedSources) == 0 || maxAge <= 0 {
		return fmt.Errorf("market observation policy is incomplete")
	}
	if !strings.EqualFold(strings.TrimSpace(observation.Symbol), strings.TrimSpace(symbol)) || strings.TrimSpace(symbol) == "" {
		return fmt.Errorf("market observation symbol mismatch")
	}
	source := strings.ToLower(strings.TrimSpace(observation.Source))
	if source == "" || (!allowNonProductionSources && (strings.Contains(source, "unknown") || strings.Contains(source, "test") || strings.Contains(source, "synthetic") || strings.Contains(source, "fixture"))) {
		return fmt.Errorf("market observation source is not genuine")
	}
	allowed := false
	for _, candidate := range allowedSources {
		if strings.EqualFold(source, strings.TrimSpace(candidate)) && strings.TrimSpace(candidate) != "" {
			allowed = true
			break
		}
	}
	if !allowed {
		return fmt.Errorf("market observation source %q is not allowed", source)
	}
	if strings.TrimSpace(observation.Mode) == "" || strings.TrimSpace(observation.Provenance) == "" {
		return fmt.Errorf("market observation mode and provenance classification are required")
	}
	if requireQuoteSides || strings.EqualFold(strings.TrimSpace(observation.Mode), "QUOTE") {
		if err := observation.ValidateQuoteTemporal(asOf, maxAge, MaxQuoteClockSkew); err != nil {
			return err
		}
	} else {
		if observation.ProviderAt.IsZero() || observation.ReceivedAt.IsZero() || asOf.IsZero() || observation.ReceivedAt.Before(observation.ProviderAt) {
			return fmt.Errorf("market observation provenance timestamps are incomplete or invalid")
		}
		if observation.ProviderAt.After(asOf) || observation.ReceivedAt.After(asOf) {
			return fmt.Errorf("market observation is from the future")
		}
		if age := asOf.Sub(observation.ProviderAt); age < 0 || age > maxAge || asOf.Sub(observation.ReceivedAt) > maxAge {
			return fmt.Errorf("market observation is stale")
		}
	}
	if timeframe != "" && strings.TrimSpace(observation.Timeframe) != timeframe {
		return fmt.Errorf("market observation timeframe mismatch")
	}
	if !validObservationPrice(observation.Last) || (requireQuoteSides && (!validObservationPrice(observation.Bid) || !validObservationPrice(observation.Ask) || observation.Bid > observation.Ask)) {
		return fmt.Errorf("market observation prices are invalid")
	}
	return nil
}

// ValidateQuoteTemporal requires separate bid/ask and last-trade timestamps.
// Small cross-clock skew is retained without rewriting source timestamps, but
// the observation cannot be used until all source and receipt times are
// available at asOf.
func (observation EconomicObservation) ValidateQuoteTemporal(asOf time.Time, maxAge, maxClockSkew time.Duration) error {
	return ValidateQuoteTemporal(observation.ProviderAt, observation.LastProviderAt, observation.ReceivedAt, asOf, maxAge, maxClockSkew)
}

func ValidateQuoteTemporal(providerAt, lastProviderAt, receivedAt, asOf time.Time, maxAge, maxClockSkew time.Duration) error {
	if providerAt.IsZero() || lastProviderAt.IsZero() || receivedAt.IsZero() || asOf.IsZero() || maxAge <= 0 || maxClockSkew < 0 {
		return fmt.Errorf("quote provenance timestamps or temporal policy are incomplete")
	}
	for _, timestamp := range []time.Time{providerAt, lastProviderAt} {
		if timestamp.After(receivedAt) && timestamp.Sub(receivedAt) > maxClockSkew {
			return fmt.Errorf("%w: provider timestamp exceeds receipt clock by %s", ErrProviderClockSkewExceeded, timestamp.Sub(receivedAt))
		}
	}
	availableAt := receivedAt
	if providerAt.After(availableAt) {
		availableAt = providerAt
	}
	if lastProviderAt.After(availableAt) {
		availableAt = lastProviderAt
	}
	if asOf.Before(availableAt) {
		return ErrQuoteNotYetAvailable
	}
	if asOf.Sub(providerAt) > maxAge || asOf.Sub(lastProviderAt) > maxAge {
		return ErrQuoteStale
	}
	return nil
}

func validObservationPrice(value float64) bool {
	return value > 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}
