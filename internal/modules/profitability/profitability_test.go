package profitability

import (
	"math"
	"testing"
	"time"
)

func TestClassifyMarketRegime(t *testing.T) {
	now := time.Date(2026, 6, 10, 14, 30, 0, 0, time.UTC)

	t.Run("risk on", func(t *testing.T) {
		snapshot := ClassifyMarketRegime(MarketRegimeInput{
			AsOfUTC: now,
			Assets: map[string]AssetState{
				"SPY": {Trend: TrendUp, AboveMA20: true, AboveMA50: true},
				"QQQ": {Trend: TrendUp, RelativeToSPY: 0.8, AboveMA20: true, AboveMA50: true},
				"IWM": {Trend: TrendUp},
				"VIX": {Trend: TrendDown},
				"XLF": {Trend: TrendFlat},
			},
		})

		if snapshot.PrimaryRegime != RegimeRiskOn {
			t.Fatalf("primary regime = %q, want %q", snapshot.PrimaryRegime, RegimeRiskOn)
		}
		if snapshot.Confidence <= 0.6 {
			t.Fatalf("confidence = %.2f, want above 0.60", snapshot.Confidence)
		}
	})

	t.Run("missing proxies are explicit", func(t *testing.T) {
		snapshot := ClassifyMarketRegime(MarketRegimeInput{
			AsOfUTC: now,
			Assets:  map[string]AssetState{"SPY": {Trend: TrendUp}},
		})

		if len(snapshot.MissingInputs) == 0 {
			t.Fatal("expected missing inputs")
		}
		if snapshot.PrimaryRegime == "" {
			t.Fatal("expected explicit regime")
		}
	})
}

func TestEvaluateCrossAssetConfirmation(t *testing.T) {
	result := EvaluateCrossAssetConfirmation(CrossAssetInput{
		MacroEventID: "event-1",
		PlaybookKey:  "cpi_rates_shock",
		Expected: map[string]Direction{
			"QQQ": DirectionDown,
			"TLT": DirectionDown,
			"SPY": DirectionDown,
		},
		Observed: map[string]Direction{
			"QQQ": DirectionDown,
			"TLT": DirectionUp,
			"SPY": DirectionDown,
		},
	})

	if result.Verdict != CrossAssetConflicted {
		t.Fatalf("verdict = %q, want %q", result.Verdict, CrossAssetConflicted)
	}
	if len(result.Conflicts) != 1 {
		t.Fatalf("conflicts = %v, want one conflict", result.Conflicts)
	}
}

func TestNormalizeEconomicCalendarEvent(t *testing.T) {
	releaseTime := time.Date(2026, 6, 10, 12, 30, 0, 0, time.UTC)
	actual := 3.2
	forecast := 2.8
	event, result := NormalizeEconomicCalendarEvent(CalendarEventInput{
		Provider:        "fixture",
		ProviderEventID: "cpi-2026-06",
		EventType:       "US_CPI_HEADLINE_YOY",
		Country:         "US",
		ReleaseTimeUTC:  releaseTime,
		Actual:          &actual,
		Forecast:        &forecast,
		Importance:      "high",
		NowUTC:          releaseTime.Add(2 * time.Minute),
	})

	if !result.Valid {
		t.Fatalf("calendar event invalid: %s", result.Reason)
	}
	if math.Abs(event.SurpriseValue-0.4) > 0.0001 {
		t.Fatalf("surprise value = %.4f, want 0.4", event.SurpriseValue)
	}
	if event.Direction != CalendarDirectionHawkishRates {
		t.Fatalf("direction = %q, want %q", event.Direction, CalendarDirectionHawkishRates)
	}
}

func TestDetectConfounders(t *testing.T) {
	primaryTime := time.Date(2026, 6, 10, 12, 30, 0, 0, time.UTC)
	links := DetectConfounders(ConfounderInput{
		PrimaryEventID: "event-1",
		PrimaryTimeUTC: primaryTime,
		PrimarySymbols: []string{"QQQ"},
		NearbyEvents: []ConfounderEvent{
			{
				ID:              "headline-1",
				Type:            ConfounderMegaCapEarnings,
				AffectedSymbols: []string{"QQQ"},
				Headline:        "Mega-cap AI leader cuts guidance",
				EventTimeUTC:    primaryTime.Add(2 * time.Minute),
				Severity:        SeverityHigh,
				Confidence:      0.9,
			},
		},
	})

	if len(links) != 1 {
		t.Fatalf("links = %d, want 1", len(links))
	}
	if links[0].Impact != ConfounderBlocksTrade {
		t.Fatalf("impact = %q, want %q", links[0].Impact, ConfounderBlocksTrade)
	}
}

func TestEvaluateExecutionQuality(t *testing.T) {
	eventTime := time.Date(2026, 6, 10, 12, 30, 0, 0, time.UTC)
	snapshot := EvaluateExecutionQuality(ExecutionQualityInput{
		Symbol:                     "QQQ",
		AsOfUTC:                    eventTime.Add(30 * time.Second),
		EventTimeUTC:               eventTime,
		SpreadPercent:              ptr(0.03),
		SlippageEstimatePercent:    ptr(0.04),
		VolumeRatio:                1.2,
		MarketDataFresh:            true,
		BrokerAvailable:            true,
		EventNoTradeDelaySeconds:   180,
		MaxSpreadPercent:           0.15,
		MaxSlippageEstimatePercent: 0.25,
	})

	if snapshot.Verdict != ExecutionBlocked {
		t.Fatalf("verdict = %q, want %q", snapshot.Verdict, ExecutionBlocked)
	}
	if len(snapshot.Reasons) == 0 {
		t.Fatal("expected blocked reason")
	}
}

func TestRecommendPositionSize(t *testing.T) {
	recommendation := RecommendPositionSize(PositionSizingInput{
		CandidateID:      "candidate-1",
		Symbol:           "QQQ",
		AccountEquity:    100_000,
		EntryPrice:       100,
		StopPrice:        98,
		RequestedRiskPct: 0.005,
		MaxRiskPct:       0.005,
		MaxDailyLossPct:  0.01,
		MaxWeeklyLossPct: 0.02,
		Confidence:       0.8,
		MarketRegime:     RegimeHighVolatility,
	})

	if recommendation.Verdict != PositionReduced {
		t.Fatalf("verdict = %q, want %q", recommendation.Verdict, PositionReduced)
	}
	if recommendation.PositionSize <= 0 {
		t.Fatalf("position size = %.2f, want positive", recommendation.PositionSize)
	}
	if recommendation.AdjustedRiskPercent >= recommendation.RiskPercent {
		t.Fatalf("adjusted risk = %.4f, want below requested %.4f", recommendation.AdjustedRiskPercent, recommendation.RiskPercent)
	}
}

func TestEvaluateStrategyPlaybook(t *testing.T) {
	result := EvaluateStrategyPlaybook(StrategyPlaybookInput{
		EventType:         "US_CPI_HEADLINE_YOY",
		Symbol:            "QQQ",
		FundamentalScore:  82,
		TechnicalScore:    78,
		Regime:            RegimeRatesDominant,
		CrossAssetVerdict: CrossAssetConfirmed,
		ExecutionVerdict:  ExecutionAcceptable,
		PositionVerdict:   PositionAllowed,
		MinutesAfterEvent: 8,
		BacktestStatus:    "paper_validated",
	})

	if result.PlaybookKey != "cpi_rates_shock" || result.Result != StrategyMatchedAllowed {
		t.Fatalf("playbook result = %q/%q, want cpi_rates_shock/%q", result.PlaybookKey, result.Result, StrategyMatchedAllowed)
	}
}

func TestBuildWalkAwayDecision(t *testing.T) {
	decisions := BuildWalkAwayDecisions(WalkAwayInput{
		EventID: "event-1",
		Symbol:  "QQQ",
		CrossAsset: CrossAssetResult{
			Verdict:   CrossAssetConflicted,
			Conflicts: []string{"TLT moved opposite expected direction"},
		},
		Strategy: StrategyPlaybookResult{Result: StrategyNoMatch, FailedRules: []string{"no strategy matched"}},
	})

	if len(decisions) != 2 {
		t.Fatalf("decisions = %d, want 2", len(decisions))
	}
	if !HasBlockingWalkAway(decisions) {
		t.Fatal("expected blocking walk-away decision")
	}
}

func TestEvaluateCandidateGateBlocksHardVetoes(t *testing.T) {
	decision := EvaluateCandidateGate(CandidateGateInput{
		Regime: MarketRegimeSnapshot{PrimaryRegime: RegimeRiskOn},
		CrossAsset: CrossAssetResult{
			Verdict:   CrossAssetConflicted,
			Conflicts: []string{"TLT moved opposite expected direction"},
		},
		Execution: ExecutionQualitySnapshot{Verdict: ExecutionGood},
		Position:  PositionSizeRecommendation{Verdict: PositionAllowed},
		Strategy:  StrategyPlaybookResult{Result: StrategyMatchedAllowed},
	})

	if decision.Allowed {
		t.Fatal("expected conflicted cross-asset confirmation to block candidate")
	}
	if len(decision.Blockers) == 0 {
		t.Fatal("expected blocker reason")
	}
}

func TestReviewTradeCalculatesExcursions(t *testing.T) {
	review := ReviewTrade(TradeReviewInput{
		CandidateID: "candidate-1",
		Symbol:      "QQQ",
		StrategyKey: "cpi_rates_shock",
		EntryPrice:  100,
		StopPrice:   98,
		TargetPrice: 104,
		ExitPrice:   103,
		Candles: []PriceCandle{
			{High: 101, Low: 99},
			{High: 105, Low: 97},
		},
	})

	if review.FinalR != 1.5 {
		t.Fatalf("final R = %.2f, want 1.50", review.FinalR)
	}
	if review.MFER != 2.5 || review.MAER != -1.5 {
		t.Fatalf("MFE/MAE = %.2f/%.2f, want 2.50/-1.50", review.MFER, review.MAER)
	}
}

func TestBuildPerformanceDashboard(t *testing.T) {
	dashboard := BuildPerformanceDashboard(PerformanceInput{
		EventsAnalyzed: 4,
		Candidates:     2,
		WalkAways:      2,
		Reviews: []TradeReview{
			{StrategyKey: "cpi_rates_shock", FinalR: 1.5, Outcome: TradeOutcomeWin},
			{StrategyKey: "cpi_rates_shock", FinalR: -1.0, Outcome: TradeOutcomeLoss},
		},
	})

	if dashboard.EventFunnel.CandidateRate != 0.5 {
		t.Fatalf("candidate rate = %.2f, want 0.50", dashboard.EventFunnel.CandidateRate)
	}
	if dashboard.StrategyPerformance[0].AverageR != 0.25 {
		t.Fatalf("avg R = %.2f, want 0.25", dashboard.StrategyPerformance[0].AverageR)
	}
}

func TestRunRiskSimulation(t *testing.T) {
	result := RunRiskSimulation(RiskSimulationInput{
		StrategyKey:     "cpi_rates_shock",
		RMultiples:      []float64{1.5, -1, -1, 2, 0.5, -1, 1},
		SimulationCount: 100,
		RiskPerTradePct: 0.005,
		MinSampleSize:   5,
	})

	if result.Verdict == SimulationInsufficientSample {
		t.Fatalf("verdict = %q, want simulated verdict", result.Verdict)
	}
	if result.MaxLossStreak < 2 {
		t.Fatalf("max loss streak = %d, want at least 2", result.MaxLossStreak)
	}
}

func ptr(v float64) *float64 {
	return &v
}
