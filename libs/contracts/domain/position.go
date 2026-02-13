package domain

import "time"

// Position represents a current holding
type Position struct {
	Symbol        string    `json:"symbol"`
	Quantity      int       `json:"quantity"`
	AvgEntryPrice float64   `json:"avg_entry_price"`
	CurrentPrice  float64   `json:"current_price"`
	UnrealizedPL  float64   `json:"unrealized_pl"`
	RealizedPL    float64   `json:"realized_pl"`
	OpenedAt      time.Time `json:"opened_at"`
	LastUpdated   time.Time `json:"last_updated"`
}

// Portfolio represents all holdings and cash
type Portfolio struct {
	AccountID   string     `json:"account_id"`
	Cash        float64    `json:"cash"`
	BuyingPower float64    `json:"buying_power"`
	Positions   []Position `json:"positions"`
	TotalValue  float64    `json:"total_value"`
	TotalPL     float64    `json:"total_pl"`
	LastUpdated time.Time  `json:"last_updated"`
}
