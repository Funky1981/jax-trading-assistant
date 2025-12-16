package utcp

type PositionSizeInput struct {
	AccountSize float64 `json:"accountSize"`
	RiskPercent float64 `json:"riskPercent"`
	Entry       float64 `json:"entry"`
	Stop        float64 `json:"stop"`
}

type PositionSizeOutput struct {
	PositionSize int     `json:"positionSize"`
	RiskPerUnit  float64 `json:"riskPerUnit"`
	TotalRisk    float64 `json:"totalRisk"`
}

type RMultipleInput struct {
	Entry  float64 `json:"entry"`
	Stop   float64 `json:"stop"`
	Target float64 `json:"target"`
}

type RMultipleOutput struct {
	RMultiple float64 `json:"rMultiple"`
}
