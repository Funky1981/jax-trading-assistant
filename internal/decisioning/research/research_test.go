package research

import "testing"

func TestValidateBacktestEvidenceRules(t *testing.T) {
	tests := []struct {
		name         string
		mutate       func(*BacktestEvidence)
		wantValid    bool
		wantMax      PromotionState
		wantDecision PromotionState
	}{
		{
			name: "missing dataset hash is invalid and capped weak",
			mutate: func(e *BacktestEvidence) {
				e.DatasetHash = ""
			},
			wantValid:    false,
			wantMax:      PromotionBacktestedWeak,
			wantDecision: PromotionBacktestedWeak,
		},
		{
			name: "missing costs and slippage are invalid and capped weak",
			mutate: func(e *BacktestEvidence) {
				e.SlippageModel = SlippageModel{}
				e.FeesModel = FeesModel{}
			},
			wantValid:    false,
			wantMax:      PromotionBacktestedWeak,
			wantDecision: PromotionBacktestedWeak,
		},
		{
			name: "missing out of sample evidence is invalid and capped weak",
			mutate: func(e *BacktestEvidence) {
				e.OutOfSamplePeriod = DateRange{}
			},
			wantValid:    false,
			wantMax:      PromotionBacktestedWeak,
			wantDecision: PromotionBacktestedWeak,
		},
		{
			name:         "complete evidence can be promising",
			wantValid:    true,
			wantMax:      PromotionBacktestedPromising,
			wantDecision: PromotionBacktestedPromising,
		},
		{
			name: "paper ready requires promising evidence plus risk rules",
			mutate: func(e *BacktestEvidence) {
				e.PromotionDecision = PromotionPaperReady
				e.RiskRules = []string{"risk/reward >= 2:1", "defined stop and invalidation"}
			},
			wantValid:    true,
			wantMax:      PromotionPaperReady,
			wantDecision: PromotionPaperReady,
		},
		{
			name: "live ready is rejected",
			mutate: func(e *BacktestEvidence) {
				e.PromotionDecision = disallowedLiveReady
			},
			wantValid:    false,
			wantMax:      PromotionBacktestedPromising,
			wantDecision: PromotionBacktestedPromising,
		},
		{
			name: "weak sample size caps promotion",
			mutate: func(e *BacktestEvidence) {
				e.SampleSize = 8
			},
			wantValid:    true,
			wantMax:      PromotionBacktestedWeak,
			wantDecision: PromotionBacktestedWeak,
		},
		{
			name: "missing failure modes is invalid and capped weak",
			mutate: func(e *BacktestEvidence) {
				e.FailureModes = nil
			},
			wantValid:    false,
			wantMax:      PromotionBacktestedWeak,
			wantDecision: PromotionBacktestedWeak,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evidence := validBacktestEvidence()
			if tt.mutate != nil {
				tt.mutate(&evidence)
			}

			got := ValidateBacktestEvidence(evidence)

			if got.IsValid != tt.wantValid {
				t.Fatalf("is_valid = %v, want %v; errors=%v warnings=%v", got.IsValid, tt.wantValid, got.ValidationErrors, got.ValidationWarnings)
			}
			if got.MaxAllowedPromotionState != tt.wantMax {
				t.Fatalf("max_allowed_promotion_state = %s, want %s", got.MaxAllowedPromotionState, tt.wantMax)
			}
			if got.PromotionDecision != tt.wantDecision {
				t.Fatalf("promotion_decision = %s, want %s", got.PromotionDecision, tt.wantDecision)
			}
		})
	}
}

func validBacktestEvidence() BacktestEvidence {
	return BacktestEvidence{
		HypothesisID:       "hyp_swing_001",
		SetupFamily:        "post_earnings_drift_continuation",
		DatasetID:          "earnings_daily_2018_2025_v1",
		DatasetHash:        "sha256:1234567890abcdef",
		DateRange:          DateRange{Start: "2018-01-01", End: "2025-12-31"},
		InstrumentUniverse: []string{"US_large_cap", "UK_large_cap"},
		Benchmark:          "SPY",
		Assumptions: BacktestAssumptions{
			Execution:       "next_day_open",
			PositionSizing:  "fixed_fractional",
			MaxRiskPerTrade: 0.01,
		},
		SlippageModel:     SlippageModel{Type: "bps", Value: 10},
		FeesModel:         FeesModel{Commission: "fixed", SpreadAssumption: "included"},
		InSamplePeriod:    DateRange{Start: "2018-01-01", End: "2022-12-31"},
		OutOfSamplePeriod: DateRange{Start: "2023-01-01", End: "2025-12-31"},
		PerformanceMetrics: PerformanceMetrics{
			TotalReturn:      0.30,
			AnnualisedReturn: 0.08,
			Sharpe:           1.2,
			Sortino:          1.5,
			ProfitFactor:     1.6,
		},
		DrawdownMetrics:   DrawdownMetrics{MaxDrawdown: -0.12, AverageDrawdown: -0.04},
		WinRate:           0.56,
		AverageWinLoss:    1.4,
		Expectancy:        0.08,
		SampleSize:        84,
		FailureModes:      []string{"overfitting", "macro regime change"},
		PromotionDecision: PromotionBacktestedPromising,
	}
}
