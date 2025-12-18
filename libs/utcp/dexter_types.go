package utcp

const (
	DexterProviderID           = "dexter"
	ToolDexterResearchCompany  = "dexter.research_company"
	ToolDexterCompareCompanies = "dexter.compare_companies"
)

type ResearchBundle struct {
	Ticker      string             `json:"ticker"`
	Summary     string             `json:"summary"`
	KeyPoints   []string           `json:"key_points"`
	Metrics     map[string]float64 `json:"metrics"`
	RawMarkdown string             `json:"raw_markdown"`
}

type ComparisonItem struct {
	Ticker string   `json:"ticker"`
	Thesis string   `json:"thesis"`
	Notes  []string `json:"notes"`
}

type ComparisonResult struct {
	ComparisonAxis string           `json:"comparison_axis"`
	Items          []ComparisonItem `json:"items"`
}
