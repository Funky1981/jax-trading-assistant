package strategytypes

import (
	"context"
	"time"
)

type StrategyType interface {
	Metadata() StrategyMetadata
	Validate(params map[string]any) error
	Generate(ctx context.Context, input StrategyInput) ([]Signal, error)
}

type StrategyMetadata struct {
	StrategyID     string         `json:"strategyId"`
	Name           string         `json:"name"`
	Description    string         `json:"description"`
	RequiredInputs RequiredInputs `json:"requiredInputs"`
	Parameters     []ParameterDef `json:"parameters"`
}

type RequiredInputs struct {
	Candles      []string `json:"candles"`
	NeedsEarnings bool    `json:"needsEarnings"`
	NeedsNews     bool    `json:"needsNews"`
}

type ParameterDef struct {
	Key         string   `json:"key"`
	Type        string   `json:"type"`
	Default     any      `json:"default"`
	Minimum     *float64 `json:"min,omitempty"`
	Maximum     *float64 `json:"max,omitempty"`
	Description string   `json:"description,omitempty"`
}

type StrategyInput struct {
	Symbol      string              `json:"symbol"`
	SessionDate time.Time           `json:"sessionDate"`
	Timezone    string              `json:"timezone"`
	Candles     map[string][]Candle `json:"candles"`
	Earnings    []EarningsEvent     `json:"earnings,omitempty"`
	News        []NewsEvent         `json:"news,omitempty"`
	Parameters  map[string]any      `json:"parameters"`
	FlattenTime string              `json:"flattenTime"`
}

type Candle struct {
	Timestamp time.Time `json:"timestamp"`
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"`
	Volume    float64   `json:"volume"`
}

type EarningsEvent struct {
	Timestamp     time.Time `json:"timestamp"`
	SurprisePct   float64   `json:"surprisePct"`
	Guidance      string    `json:"guidance"`
	PreviousClose float64   `json:"previousClose"`
}

type NewsEvent struct {
	Timestamp   time.Time `json:"timestamp"`
	Category    string    `json:"category"`
	Materiality string    `json:"materiality"`
	Sentiment   string    `json:"sentiment"`
}

type Signal struct {
	Symbol      string    `json:"symbol"`
	StrategyID  string    `json:"strategyId"`
	Direction   string    `json:"direction"`
	EntryPrice  float64   `json:"entryPrice"`
	StopLoss    float64   `json:"stopLoss"`
	TakeProfit  float64   `json:"takeProfit"`
	GeneratedAt time.Time `json:"generatedAt"`
	Reason      string    `json:"reason"`
}
