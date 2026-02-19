package domain

type RiskResult struct {
	PositionSize int     `json:"positionSize"`
	RiskPerUnit  float64 `json:"riskPerUnit"`
	TotalRisk    float64 `json:"totalRisk"`
	RMultiple    float64 `json:"rMultiple"`
}
