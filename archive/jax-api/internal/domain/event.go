package domain

import "time"

type EventType string

const (
	EventEarningsSurprise EventType = "earnings_surprise"
	EventGapOpen          EventType = "gap_open"
	EventVolumeSpike      EventType = "volume_spike"
)

type Event struct {
	ID      string         `json:"id"`
	Symbol  string         `json:"symbol"`
	Type    EventType      `json:"type"`
	Time    time.Time      `json:"time"`
	Payload map[string]any `json:"payload"`
}
