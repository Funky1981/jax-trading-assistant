package utcp

import "time"

const (
	StorageProviderID      = "storage"
	ToolStorageSaveEvent   = "storage.save_event"
	ToolStorageSaveTrade   = "storage.save_trade"
	ToolStorageGetTrade    = "storage.get_trade"
	ToolStorageListTrades  = "storage.list_trades"
	DefaultListTradesLimit = 100
)

type StoredEvent struct {
	ID      string         `json:"id"`
	Symbol  string         `json:"symbol"`
	Type    string         `json:"type"`
	Time    time.Time      `json:"time"`
	Payload map[string]any `json:"payload"`
}

type StoredRisk struct {
	PositionSize int     `json:"positionSize"`
	RiskPerUnit  float64 `json:"riskPerUnit,omitempty"`
	TotalRisk    float64 `json:"totalRisk"`
	RMultiple    float64 `json:"rMultiple,omitempty"`
}

type StoredTrade struct {
	ID         string    `json:"id"`
	Symbol     string    `json:"symbol"`
	Direction  string    `json:"direction"`
	Entry      float64   `json:"entry"`
	Stop       float64   `json:"stop"`
	Targets    []float64 `json:"targets"`
	EventID    string    `json:"eventId,omitempty"`
	StrategyID string    `json:"strategyId"`
	Notes      string    `json:"notes,omitempty"`
	CreatedAt  time.Time `json:"createdAt,omitempty"`
}

type SaveEventInput struct {
	Event StoredEvent `json:"event"`
}

type SaveEventOutput struct{}

type SaveTradeInput struct {
	Trade StoredTrade  `json:"trade"`
	Risk  *StoredRisk  `json:"risk,omitempty"`
	Event *StoredEvent `json:"event,omitempty"`
}

type SaveTradeOutput struct{}

type GetTradeInput struct {
	ID string `json:"id"`
}

type GetTradeOutput struct {
	Trade StoredTrade  `json:"trade"`
	Risk  *StoredRisk  `json:"risk,omitempty"`
	Event *StoredEvent `json:"event,omitempty"`
}

type ListTradesInput struct {
	Symbol     string `json:"symbol,omitempty"`
	StrategyID string `json:"strategyId,omitempty"`
	Limit      int    `json:"limit,omitempty"`
}

type ListTradesOutput struct {
	Trades []GetTradeOutput `json:"trades"`
}
