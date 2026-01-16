package ingest

import "time"

const (
	ObservationEarnings      = "earnings"
	ObservationNewsHeadline  = "news_headline"
	ObservationUnusualVolume = "unusual_volume"
	ObservationPriceGap      = "price_gap"

	KindMarketEvent = "market_event"
	KindSignal      = "signal"
)

type DexterObservation struct {
	ID             string    `json:"id,omitempty"`
	Type           string    `json:"type"`
	Symbol         string    `json:"symbol,omitempty"`
	Headline       string    `json:"headline,omitempty"`
	ImpactEstimate float64   `json:"impact_estimate,omitempty"`
	Confidence     float64   `json:"confidence,omitempty"`
	Tags           []string  `json:"tags,omitempty"`
	TS             time.Time `json:"ts,omitempty"`
	VolumeMultiple float64   `json:"volume_multiple,omitempty"`
	GapPercent     float64   `json:"gap_percent,omitempty"`
	Bookmarked     bool      `json:"bookmarked,omitempty"`
	SourceRef      string    `json:"source_ref,omitempty"`
}

type DexterPayload struct {
	GeneratedAt  time.Time          `json:"generated_at,omitempty"`
	Observations []DexterObservation `json:"observations"`
}

type MarketEvent struct {
	TS             time.Time
	EventType      string
	Symbol         string
	Tags           []string
	Summary        string
	ImpactEstimate float64
	Confidence     float64
	Headline       string
	SourceRef      string
	Bookmarked     bool
}

type Signal struct {
	TS             time.Time
	SignalType     string
	Symbol         string
	Tags           []string
	Summary        string
	ImpactEstimate float64
	Confidence     float64
	VolumeMultiple float64
	GapPercent     float64
	SourceRef      string
	Bookmarked     bool
}

type NormalizedObservation struct {
	Kind   string
	Event  *MarketEvent
	Signal *Signal
}

type RetentionConfig struct {
	SignificanceThreshold float64
}

type RetentionResult struct {
	Retained int
	Skipped  int
}
