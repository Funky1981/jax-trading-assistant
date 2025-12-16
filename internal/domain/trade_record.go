package domain

import "time"

type TradeRecord struct {
	Setup     TradeSetup `json:"setup"`
	Risk      RiskResult `json:"risk"`
	Event     *Event     `json:"event,omitempty"`
	CreatedAt time.Time  `json:"createdAt,omitempty"`
}
