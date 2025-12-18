package domain

type ResearchBundle struct {
	Ticker      string             `json:"ticker"`
	Summary     string             `json:"summary"`
	KeyPoints   []string           `json:"key_points"`
	Metrics     map[string]float64 `json:"metrics"`
	RawMarkdown string             `json:"raw_markdown"`
}
