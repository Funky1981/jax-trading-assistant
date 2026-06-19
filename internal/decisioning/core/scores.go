package core

type Scores struct {
	ConfidenceScore   float64 `json:"confidence_score"`
	ClarityScore      float64 `json:"clarity_score"`
	EdgeScore         float64 `json:"edge_score"`
	ConflictScore     float64 `json:"conflict_score"`
	RiskScore         float64 `json:"risk_score"`
	ConfirmationScore float64 `json:"confirmation_score,omitempty"`
	TimingScore       float64 `json:"timing_score,omitempty"`
}

func (s Scores) withConfidence() Scores {
	s.ClarityScore = clampScore(s.ClarityScore)
	s.EdgeScore = clampScore(s.EdgeScore)
	s.ConflictScore = clampScore(s.ConflictScore)
	s.RiskScore = clampScore(s.RiskScore)
	s.ConfirmationScore = clampScore(s.ConfirmationScore)
	s.TimingScore = clampScore(s.TimingScore)
	if s.ConfidenceScore == 0 {
		rejectionConfidence := maxFloat(s.ConflictScore, s.RiskScore, 1-s.EdgeScore, 1-s.ClarityScore)
		candidateConfidence := (s.ClarityScore + s.EdgeScore + (1 - s.ConflictScore) + (1 - s.RiskScore)) / 4
		s.ConfidenceScore = clampScore(maxFloat(rejectionConfidence, candidateConfidence))
	} else {
		s.ConfidenceScore = clampScore(s.ConfidenceScore)
	}
	return s
}

func clampScore(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func maxFloat(values ...float64) float64 {
	if len(values) == 0 {
		return 0
	}
	max := values[0]
	for _, value := range values[1:] {
		if value > max {
			max = value
		}
	}
	return max
}
