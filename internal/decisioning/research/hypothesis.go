package research

// ResearchHypothesis describes the setup-family claim being tested before
// promotion toward paper trading can be considered.
type ResearchHypothesis struct {
	HypothesisID         string   `json:"hypothesis_id"`
	SetupFamily          string   `json:"setup_family"`
	Claim                string   `json:"claim"`
	TargetAssets         []string `json:"target_assets"`
	HoldingPeriod        string   `json:"holding_period"`
	EntryRule            string   `json:"entry_rule"`
	ExitRule             string   `json:"exit_rule"`
	RiskRule             string   `json:"risk_rule"`
	ExpectedFailureModes []string `json:"expected_failure_modes"`
}

func (h ResearchHypothesis) HasPaperReadyDefinition() bool {
	return h.HypothesisID != "" &&
		h.SetupFamily != "" &&
		h.Claim != "" &&
		len(h.TargetAssets) > 0 &&
		h.HoldingPeriod != "" &&
		h.EntryRule != "" &&
		h.ExitRule != "" &&
		h.RiskRule != "" &&
		len(h.ExpectedFailureModes) > 0
}
