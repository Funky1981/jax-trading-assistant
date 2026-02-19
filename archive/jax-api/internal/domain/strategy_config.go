package domain

type StrategyConfig struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	EventTypes     []string `json:"eventTypes"`
	MinRR          float64  `json:"minRR"`
	MaxRiskPercent float64  `json:"maxRiskPercent"`

	EntryRule  string `json:"entryRule"`
	StopRule   string `json:"stopRule"`
	TargetRule string `json:"targetRule"`
}
