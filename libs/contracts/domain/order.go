package domain

import "time"

// Order represents a trade execution order
type Order struct {
	ID              string     `json:"id"`
	Symbol          string     `json:"symbol"`
	Type            string     `json:"type"` // "market", "limit", "stop"
	Side            string     `json:"side"` // "buy", "sell"
	Quantity        int        `json:"quantity"`
	Price           float64    `json:"price,omitempty"`
	StopPrice       float64    `json:"stop_price,omitempty"`
	TimeInForce     string     `json:"time_in_force"` // "day", "gtc", "ioc"
	Status          string     `json:"status"`
	SubmittedAt     time.Time  `json:"submitted_at"`
	FilledAt        *time.Time `json:"filled_at,omitempty"`
	FilledQuantity  int        `json:"filled_quantity"`
	FilledPrice     float64    `json:"filled_price,omitempty"`
	Commission      float64    `json:"commission,omitempty"`
	RejectionReason string     `json:"rejection_reason,omitempty"`
}
