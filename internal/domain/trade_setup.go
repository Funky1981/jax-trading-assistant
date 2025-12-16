package domain

type TradeDirection string

const (
	Long  TradeDirection = "long"
	Short TradeDirection = "short"
)

type TradeSetup struct {
	ID         string         `json:"id"`
	Symbol     string         `json:"symbol"`
	Direction  TradeDirection `json:"direction"`
	Entry      float64        `json:"entry"`
	Stop       float64        `json:"stop"`
	Targets    []float64      `json:"targets"`
	EventID    string         `json:"eventId"`
	StrategyID string         `json:"strategyId"`
	Notes      string         `json:"notes"`

	Research *ResearchBundle `json:"research,omitempty"`
}
