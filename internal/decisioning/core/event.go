package core

import "time"

type Event struct {
	EventID            string    `json:"event_id"`
	SourceType         string    `json:"source_type"`
	SourceURL          string    `json:"source_url,omitempty"`
	ReceivedAt         time.Time `json:"received_at"`
	Headline           string    `json:"headline"`
	Summary            string    `json:"summary"`
	EventType          string    `json:"event_type"`
	PrimaryDrivers     []string  `json:"primary_drivers"`
	ConflictingDrivers []string  `json:"conflicting_drivers"`
	AffectedAssets     []string  `json:"affected_assets"`
	AssetClasses       []string  `json:"asset_classes"`
	Geography          []string  `json:"geography"`
	TimeSensitivity    string    `json:"time_sensitivity"`
	UncertaintyNotes   []string  `json:"uncertainty_notes"`
}
