package profitability

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

func ClassifyMarketRegime(input MarketRegimeInput) MarketRegimeSnapshot {
	assets := normalizeAssetMap(input.Assets)
	snapshot := MarketRegimeSnapshot{
		AsOfUTC:       normalizedTime(input.AsOfUTC),
		PrimaryRegime: RegimeUnclear,
		Inputs:        assets,
		MissingInputs: missingAssets(assets, []string{"SPY", "QQQ", "IWM", "TLT", "GLD", "XLF", "VIX"}),
		Reasons:       []string{},
	}

	spy := assets["SPY"]
	qqq := assets["QQQ"]
	iwm := assets["IWM"]
	tlt := assets["TLT"]
	gld := assets["GLD"]
	xlf := assets["XLF"]
	vix := assets["VIX"]

	riskOnScore := boolScore(spy.Trend == TrendUp && spy.AboveMA20 && spy.AboveMA50) +
		boolScore(qqq.Trend == TrendUp && qqq.RelativeToSPY >= 0) +
		boolScore(iwm.Trend != TrendDown) +
		boolScore(vix.Trend != TrendUp) +
		boolScore(xlf.Trend != TrendDown)
	riskOffScore := boolScore(spy.Trend == TrendDown || (!spy.AboveMA20 && !spy.AboveMA50)) +
		boolScore(qqq.Trend == TrendDown) +
		boolScore(iwm.Trend == TrendDown) +
		boolScore(vix.Trend == TrendUp) +
		boolScore(tlt.Trend == TrendUp || gld.Trend == TrendUp)
	ratesScore := boolScore(tlt.Trend != TrendUnknown && qqq.Trend != TrendUnknown && tlt.Trend != qqq.Trend) +
		boolScore(math.Abs(tlt.MovePercent) >= 0.5) +
		boolScore(strings.Contains(strings.ToLower(snapshotReasonContext(input)), "cpi") || strings.Contains(strings.ToLower(snapshotReasonContext(input)), "fed"))

	switch {
	case riskOnScore >= 4:
		snapshot.PrimaryRegime = RegimeRiskOn
		snapshot.Confidence = confidenceFromScore(riskOnScore, 5, len(snapshot.MissingInputs))
		snapshot.Reasons = append(snapshot.Reasons, "broad risk assets are stable or rising")
	case riskOffScore >= 4:
		snapshot.PrimaryRegime = RegimeRiskOff
		snapshot.Confidence = confidenceFromScore(riskOffScore, 5, len(snapshot.MissingInputs))
		snapshot.Reasons = append(snapshot.Reasons, "risk assets are weakening with defensive or volatility pressure")
	case ratesScore >= 2:
		snapshot.PrimaryRegime = RegimeRatesDominant
		snapshot.Confidence = confidenceFromScore(ratesScore, 3, len(snapshot.MissingInputs))
		snapshot.Reasons = append(snapshot.Reasons, "rate-sensitive assets are leading the move")
	default:
		snapshot.Confidence = 0.25
		snapshot.Reasons = append(snapshot.Reasons, "available inputs do not support a clear market regime")
	}

	if vix.Trend == TrendUp {
		snapshot.SecondaryRegimes = append(snapshot.SecondaryRegimes, RegimeHighVolatility)
	}
	if spy.Trend == TrendDown && qqq.Trend == TrendDown && iwm.Trend == TrendDown && xlf.Trend == TrendDown {
		snapshot.SecondaryRegimes = append(snapshot.SecondaryRegimes, RegimeLiquidityStress)
	}
	return snapshot
}

func EvaluateCrossAssetConfirmation(input CrossAssetInput) CrossAssetResult {
	result := CrossAssetResult{
		MacroEventID: input.MacroEventID,
		PlaybookKey:  input.PlaybookKey,
		AsOfUTC:      normalizedTime(input.AsOfUTC),
		AssetResults: map[string]Direction{},
		Reasons:      []string{},
	}
	expected := normalizeDirectionMap(input.Expected)
	observed := normalizeDirectionMap(input.Observed)
	if len(expected) == 0 {
		result.Verdict = CrossAssetInsufficientData
		result.Reasons = append(result.Reasons, "expected asset basket is missing")
		return result
	}

	matches := 0
	for symbol, want := range expected {
		got, ok := observed[symbol]
		if !ok || got == DirectionUnknown {
			result.MissingAssets = append(result.MissingAssets, symbol)
			continue
		}
		result.AssetResults[symbol] = got
		if directionMatches(want, got) {
			matches++
			continue
		}
		if got != DirectionFlat {
			result.Conflicts = append(result.Conflicts, fmt.Sprintf("%s expected %s observed %s", symbol, want, got))
		}
	}

	result.ConfirmationScore = math.Round((float64(matches)/float64(len(expected)))*10000) / 100
	switch {
	case len(result.Conflicts) > 0:
		result.Verdict = CrossAssetConflicted
		result.Reasons = append(result.Reasons, "one or more required assets moved against the thesis")
	case len(result.MissingAssets) == len(expected):
		result.Verdict = CrossAssetInsufficientData
		result.Reasons = append(result.Reasons, "all expected confirmation assets are missing")
	case result.ConfirmationScore >= 75:
		result.Verdict = CrossAssetConfirmed
		result.Reasons = append(result.Reasons, "asset basket confirms the thesis")
	case result.ConfirmationScore >= 45:
		result.Verdict = CrossAssetPartiallyConfirmed
		result.Reasons = append(result.Reasons, "asset basket partially confirms the thesis")
	default:
		result.Verdict = CrossAssetNotConfirmed
		result.Reasons = append(result.Reasons, "asset basket did not confirm the thesis")
	}
	sort.Strings(result.Conflicts)
	sort.Strings(result.MissingAssets)
	return result
}

func NormalizeEconomicCalendarEvent(input CalendarEventInput) (CalendarEvent, ValidationResult) {
	event := CalendarEvent{CalendarEventInput: input}
	event.Provider = strings.TrimSpace(input.Provider)
	event.ProviderEventID = strings.TrimSpace(input.ProviderEventID)
	event.EventType = strings.ToUpper(strings.TrimSpace(input.EventType))
	event.Country = strings.ToUpper(strings.TrimSpace(input.Country))
	event.Importance = strings.ToLower(strings.TrimSpace(input.Importance))
	event.ReleaseTimeUTC = input.ReleaseTimeUTC.UTC()
	now := normalizedTime(input.NowUTC)

	switch {
	case event.Provider == "":
		return event, ValidationResult{Valid: false, Status: "quarantined", Reason: "provider is required"}
	case event.ProviderEventID == "":
		return event, ValidationResult{Valid: false, Status: "quarantined", Reason: "provider_event_id is required"}
	case event.EventType == "":
		return event, ValidationResult{Valid: false, Status: "quarantined", Reason: "event_type is required"}
	case event.Country == "":
		return event, ValidationResult{Valid: false, Status: "quarantined", Reason: "country is required"}
	case event.ReleaseTimeUTC.IsZero():
		return event, ValidationResult{Valid: false, Status: "quarantined", Reason: "release_time_utc is required"}
	case event.Actual == nil && now.After(event.ReleaseTimeUTC):
		return event, ValidationResult{Valid: false, Status: "quarantined", Reason: "actual is missing after release"}
	case surpriseDrivenEvent(event.EventType) && event.Forecast == nil:
		return event, ValidationResult{Valid: false, Status: "quarantined", Reason: "forecast is required for surprise-driven event"}
	}

	if event.Actual != nil && event.Forecast != nil {
		event.SurpriseValue = *event.Actual - *event.Forecast
		if *event.Forecast != 0 {
			event.SurprisePercent = event.SurpriseValue / math.Abs(*event.Forecast)
		}
	}
	event.Direction = calendarDirection(event.EventType, event.SurpriseValue)
	return event, ValidationResult{Valid: true, Status: "accepted"}
}

func DetectConfounders(input ConfounderInput) []ConfounderLink {
	links := []ConfounderLink{}
	for _, event := range input.NearbyEvents {
		if event.ID == "" {
			continue
		}
		minutes := math.Abs(event.EventTimeUTC.Sub(input.PrimaryTimeUTC).Minutes())
		overlap := overlapsAny(event.AffectedSymbols, input.PrimarySymbols)
		if minutes > 60 && !overlap {
			continue
		}
		impact := confounderImpact(event, minutes, overlap)
		links = append(links, ConfounderLink{
			EventID:           input.PrimaryEventID,
			ConfounderEventID: event.ID,
			Impact:            impact,
			Reason:            confounderReason(event, impact),
			Confounder:        event,
		})
	}
	return links
}

func EvaluateExecutionQuality(input ExecutionQualityInput) ExecutionQualitySnapshot {
	maxSpread := defaultFloat(input.MaxSpreadPercent, 0.15)
	maxSlippage := defaultFloat(input.MaxSlippageEstimatePercent, 0.25)
	minVolume := defaultFloat(input.MinVolumeRatio, 0.50)
	delaySeconds := input.EventNoTradeDelaySeconds
	if delaySeconds <= 0 {
		delaySeconds = 180
	}

	snapshot := ExecutionQualitySnapshot{
		Symbol:                  strings.ToUpper(strings.TrimSpace(input.Symbol)),
		AsOfUTC:                 normalizedTime(input.AsOfUTC),
		SpreadPercent:           input.SpreadPercent,
		SlippageEstimatePercent: input.SlippageEstimatePercent,
		VolumeOK:                input.VolumeRatio >= minVolume,
		MarketDataFresh:         input.MarketDataFresh,
		BrokerAvailable:         input.BrokerAvailable,
		EventVolatilityState:    strings.TrimSpace(input.EventVolatilityState),
		RawPayload:              input.RawPayload,
	}
	if snapshot.EventVolatilityState == "" {
		snapshot.EventVolatilityState = "normal"
	}

	switch {
	case snapshot.Symbol == "" || input.SpreadPercent == nil || input.SlippageEstimatePercent == nil:
		snapshot.Verdict = ExecutionInsufficientData
		snapshot.Reasons = append(snapshot.Reasons, "quote or symbol data is missing")
	case !input.MarketDataFresh:
		snapshot.Verdict = ExecutionBlocked
		snapshot.Reasons = append(snapshot.Reasons, "market data is stale")
	case !input.BrokerAvailable:
		snapshot.Verdict = ExecutionBlocked
		snapshot.Reasons = append(snapshot.Reasons, "broker is unavailable")
	case input.AsOfUTC.Sub(input.EventTimeUTC) >= 0 && input.AsOfUTC.Sub(input.EventTimeUTC) < time.Duration(delaySeconds)*time.Second:
		snapshot.Verdict = ExecutionBlocked
		snapshot.Reasons = append(snapshot.Reasons, "event no-trade delay is active")
	case *input.SpreadPercent > maxSpread:
		snapshot.Verdict = ExecutionBlocked
		snapshot.Reasons = append(snapshot.Reasons, "spread is too wide")
	case *input.SlippageEstimatePercent > maxSlippage:
		snapshot.Verdict = ExecutionBlocked
		snapshot.Reasons = append(snapshot.Reasons, "slippage estimate is too high")
	case !snapshot.VolumeOK:
		snapshot.Verdict = ExecutionBlocked
		snapshot.Reasons = append(snapshot.Reasons, "volume is below minimum")
	case *input.SpreadPercent > maxSpread*0.75 || *input.SlippageEstimatePercent > maxSlippage*0.75:
		snapshot.Verdict = ExecutionAcceptable
		snapshot.Reasons = append(snapshot.Reasons, "execution quality is acceptable with caution")
	default:
		snapshot.Verdict = ExecutionGood
		snapshot.Reasons = append(snapshot.Reasons, "execution quality checks passed")
	}
	return snapshot
}

func RecommendPositionSize(input PositionSizingInput) PositionSizeRecommendation {
	rec := PositionSizeRecommendation{
		CandidateID:         input.CandidateID,
		Symbol:              strings.ToUpper(strings.TrimSpace(input.Symbol)),
		AccountEquity:       input.AccountEquity,
		EntryPrice:          input.EntryPrice,
		StopPrice:           input.StopPrice,
		RiskPercent:         input.RequestedRiskPct,
		AdjustedRiskPercent: input.RequestedRiskPct,
	}
	maxRisk := defaultFloat(input.MaxRiskPct, 0.005)
	switch {
	case rec.Symbol == "" || input.AccountEquity <= 0 || input.EntryPrice <= 0:
		rec.Verdict = PositionInsufficientData
		rec.Reasons = append(rec.Reasons, "account equity, symbol, or entry price is missing")
		return rec
	case input.StopPrice <= 0 || input.EntryPrice == input.StopPrice:
		rec.Verdict = PositionBlocked
		rec.Reasons = append(rec.Reasons, "valid stop price is required")
		return rec
	case input.RequestedRiskPct > maxRisk:
		rec.Verdict = PositionBlocked
		rec.Reasons = append(rec.Reasons, "requested risk exceeds per-trade limit")
		return rec
	case input.CurrentDailyLossPct >= defaultFloat(input.MaxDailyLossPct, 0.01):
		rec.Verdict = PositionBlocked
		rec.Reasons = append(rec.Reasons, "daily loss limit hit")
		return rec
	case input.CurrentWeeklyLossPct >= defaultFloat(input.MaxWeeklyLossPct, 0.02):
		rec.Verdict = PositionBlocked
		rec.Reasons = append(rec.Reasons, "weekly loss limit hit")
		return rec
	case input.SameThemeExposureCount > 0:
		rec.Verdict = PositionBlocked
		rec.Reasons = append(rec.Reasons, "same-theme exposure limit hit")
		return rec
	case input.CorrelatedExposureCount > 0:
		rec.Verdict = PositionBlocked
		rec.Reasons = append(rec.Reasons, "correlated ETF exposure limit hit")
		return rec
	}

	if input.MarketRegime == RegimeHighVolatility || input.MarketRegime == RegimeLiquidityStress || input.MarketRegime == RegimeUnclear {
		rec.AdjustedRiskPercent *= 0.5
		rec.Adjustments = append(rec.Adjustments, "reduced for regime risk")
	}
	if input.Confidence > 0 && input.Confidence < 0.6 {
		rec.AdjustedRiskPercent *= 0.5
		rec.Adjustments = append(rec.Adjustments, "reduced for low confidence")
	}

	rec.CashRisk = math.Round(input.AccountEquity*rec.AdjustedRiskPercent*100) / 100
	rec.PositionSize = math.Round((rec.CashRisk/math.Abs(input.EntryPrice-input.StopPrice))*100) / 100
	if len(rec.Adjustments) > 0 {
		rec.Verdict = PositionReduced
		rec.Reasons = append(rec.Reasons, "position size allowed with risk reductions")
	} else {
		rec.Verdict = PositionAllowed
		rec.Reasons = append(rec.Reasons, "position size allowed")
	}
	return rec
}

func EvaluateStrategyPlaybook(input StrategyPlaybookInput) StrategyPlaybookResult {
	result := StrategyPlaybookResult{Reasons: []string{}, FailedRules: []string{}}
	eventType := strings.ToUpper(strings.TrimSpace(input.EventType))
	symbol := strings.ToUpper(strings.TrimSpace(input.Symbol))

	if strings.Contains(eventType, "CPI") && symbol == "QQQ" {
		result.PlaybookKey = "cpi_rates_shock"
	} else if strings.Contains(eventType, "NONFARM") {
		result.PlaybookKey = "nfp_rates_shock"
	} else if strings.Contains(eventType, "FOMC") || strings.Contains(eventType, "FED") {
		result.PlaybookKey = "fed_delayed_confirmation"
	}
	if result.PlaybookKey == "" {
		result.Result = StrategyNoMatch
		result.Reasons = append(result.Reasons, "no named strategy matched event and symbol")
		return result
	}
	result.Matched = true

	addFailed := func(condition bool, rule string) {
		if condition {
			result.FailedRules = append(result.FailedRules, rule)
		}
	}
	addFailed(input.FundamentalScore < 70, "fundamental score below 70")
	addFailed(input.TechnicalScore < 70, "technical score below 70")
	addFailed(input.CrossAssetVerdict != CrossAssetConfirmed, "cross-asset confirmation not confirmed")
	addFailed(input.Regime == RegimeRiskOff || input.Regime == RegimeLiquidityStress || input.Regime == RegimeUnclear, "regime conflicts with strategy")
	addFailed(input.ExecutionVerdict != ExecutionGood && input.ExecutionVerdict != ExecutionAcceptable, "execution quality not acceptable")
	addFailed(input.PositionVerdict != PositionAllowed && input.PositionVerdict != PositionReduced, "position sizing not allowed")
	addFailed(input.BacktestStatus == "" || input.BacktestStatus == "untested", "strategy lacks paper validation")
	if result.PlaybookKey == "fed_delayed_confirmation" {
		addFailed(input.MinutesAfterEvent < 15, "Fed strategy requires delayed confirmation")
	}

	switch {
	case len(result.FailedRules) == 0:
		result.Result = StrategyMatchedAllowed
		result.Reasons = append(result.Reasons, "strategy playbook matched")
	case len(result.FailedRules) <= 2:
		result.Result = StrategyMatchedWatch
		result.Reasons = append(result.Reasons, "strategy matched but requires watch-only handling")
	default:
		result.Result = StrategyMatchedBlocked
		result.Reasons = append(result.Reasons, "strategy matched but hard rules failed")
	}
	return result
}

func BuildWalkAwayDecisions(input WalkAwayInput) []WalkAwayDecision {
	decisions := []WalkAwayDecision{}
	add := func(category WalkAwayCategory, severity WalkAwaySeverity, reason string, refs map[string]any) {
		decisions = append(decisions, WalkAwayDecision{
			EventID:      input.EventID,
			Symbol:       strings.ToUpper(strings.TrimSpace(input.Symbol)),
			Category:     category,
			Severity:     severity,
			Reason:       reason,
			EvidenceRefs: refs,
		})
	}
	if input.CrossAsset.Verdict == CrossAssetConflicted {
		add(WalkAwayCrossAssetConflict, WalkAwayBlocker, "cross-asset confirmation conflicts with thesis", map[string]any{"conflicts": input.CrossAsset.Conflicts})
	}
	if input.Regime.PrimaryRegime == RegimeUnclear || input.Regime.PrimaryRegime == RegimeLiquidityStress || input.Regime.PrimaryRegime == RegimeRiskOff {
		add(WalkAwayRegimeConflict, WalkAwayBlocker, "market regime blocks or materially conflicts with setup", map[string]any{"regime": input.Regime.PrimaryRegime})
	}
	if input.Execution.Verdict == ExecutionBlocked || input.Execution.Verdict == ExecutionPoor {
		add(WalkAwayPoorLiquidity, WalkAwayBlocker, "execution quality blocks candidate", map[string]any{"execution": input.Execution.Verdict, "reasons": input.Execution.Reasons})
	}
	if input.Position.Verdict == PositionBlocked {
		add(WalkAwayRiskLimitHit, WalkAwayBlocker, "position sizing or risk limit blocks candidate", map[string]any{"reasons": input.Position.Reasons})
	}
	if input.Strategy.Result == StrategyNoMatch {
		add(WalkAwayStrategyNotMatched, WalkAwayBlocker, "no strategy playbook matched", map[string]any{"failed_rules": input.Strategy.FailedRules})
	}
	return decisions
}

func HasBlockingWalkAway(decisions []WalkAwayDecision) bool {
	for _, decision := range decisions {
		if decision.Severity == WalkAwayBlocker || decision.Severity == WalkAwayCritical {
			return true
		}
	}
	return false
}

func EvaluateCandidateGate(input CandidateGateInput) CandidateGateDecision {
	decision := CandidateGateDecision{Allowed: true}
	addBlocker := func(reason string) {
		decision.Allowed = false
		decision.Blockers = append(decision.Blockers, reason)
	}
	switch input.Regime.PrimaryRegime {
	case RegimeUnclear, RegimeRiskOff, RegimeLiquidityStress:
		addBlocker("market regime conflicts with candidate")
	}
	switch input.CrossAsset.Verdict {
	case CrossAssetConflicted, CrossAssetNotConfirmed, CrossAssetInsufficientData:
		addBlocker("cross-asset confirmation blocks candidate")
	case CrossAssetPartiallyConfirmed:
		decision.Warnings = append(decision.Warnings, "cross-asset confirmation is partial")
	}
	for _, link := range input.Confounders {
		if link.Impact == ConfounderBlocksTrade || link.Impact == ConfounderReassignsCause {
			addBlocker("blocking confounder detected")
			break
		}
	}
	switch input.Execution.Verdict {
	case ExecutionBlocked, ExecutionPoor, ExecutionInsufficientData:
		addBlocker("execution quality blocks candidate")
	}
	switch input.Position.Verdict {
	case PositionBlocked, PositionInsufficientData:
		addBlocker("position sizing or portfolio risk blocks candidate")
	}
	if input.Strategy.Result == StrategyNoMatch || input.Strategy.Result == StrategyMatchedBlocked {
		addBlocker("strategy playbook blocks candidate")
	}
	if HasBlockingWalkAway(input.WalkAways) {
		addBlocker("walk-away decision blocks candidate")
	}
	decision.Blockers = uniqueStrings(decision.Blockers)
	decision.Warnings = uniqueStrings(decision.Warnings)
	return decision
}

func ReviewTrade(input TradeReviewInput) TradeReview {
	review := TradeReview{
		CandidateID: input.CandidateID,
		Symbol:      strings.ToUpper(strings.TrimSpace(input.Symbol)),
		StrategyKey: input.StrategyKey,
		EntryPrice:  input.EntryPrice,
		ExitPrice:   input.ExitPrice,
		StopPrice:   input.StopPrice,
		TargetPrice: input.TargetPrice,
		Outcome:     input.Outcome,
	}
	risk := math.Abs(input.EntryPrice - input.StopPrice)
	if risk == 0 {
		return review
	}
	highest := input.EntryPrice
	lowest := input.EntryPrice
	for _, candle := range input.Candles {
		if candle.High > highest {
			highest = candle.High
		}
		if candle.Low < lowest {
			lowest = candle.Low
		}
	}
	review.MFER = math.Round(((highest-input.EntryPrice)/risk)*100) / 100
	review.MAER = math.Round(((lowest-input.EntryPrice)/risk)*100) / 100
	review.FinalR = math.Round(((input.ExitPrice-input.EntryPrice)/risk)*100) / 100
	if review.Outcome == "" {
		switch {
		case review.FinalR > 0:
			review.Outcome = TradeOutcomeWin
		case review.FinalR < 0:
			review.Outcome = TradeOutcomeLoss
		default:
			review.Outcome = TradeOutcomeBreakeven
		}
	}
	return review
}

func BuildPerformanceDashboard(input PerformanceInput) PerformanceDashboard {
	dashboard := PerformanceDashboard{
		EventFunnel: EventFunnelMetrics{
			EventsAnalyzed: input.EventsAnalyzed,
			Candidates:     input.Candidates,
			WalkAways:      input.WalkAways,
		},
	}
	if input.EventsAnalyzed > 0 {
		dashboard.EventFunnel.CandidateRate = float64(input.Candidates) / float64(input.EventsAnalyzed)
	}

	byStrategy := map[string][]TradeReview{}
	for _, review := range input.Reviews {
		byStrategy[review.StrategyKey] = append(byStrategy[review.StrategyKey], review)
	}
	keys := make([]string, 0, len(byStrategy))
	for key := range byStrategy {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		reviews := byStrategy[key]
		var totalR, wins float64
		for _, review := range reviews {
			totalR += review.FinalR
			if review.FinalR > 0 || review.Outcome == TradeOutcomeWin {
				wins++
			}
		}
		trades := len(reviews)
		avgR := totalR / float64(trades)
		dashboard.StrategyPerformance = append(dashboard.StrategyPerformance, StrategyPerformance{
			StrategyKey: key,
			Trades:      trades,
			WinRate:     wins / float64(trades),
			AverageR:    math.Round(avgR*100) / 100,
			Expectancy:  math.Round(avgR*100) / 100,
		})
	}
	return dashboard
}

func RunRiskSimulation(input RiskSimulationInput) RiskSimulationResult {
	minSample := input.MinSampleSize
	if minSample <= 0 {
		minSample = 30
	}
	result := RiskSimulationResult{
		StrategyKey:     input.StrategyKey,
		SimulationCount: input.SimulationCount,
		SampleSize:      len(input.RMultiples),
	}
	if len(input.RMultiples) < minSample {
		result.Verdict = SimulationInsufficientSample
		result.Warnings = append(result.Warnings, "insufficient sample size")
		return result
	}
	var total, equity, peak, maxDrawdown float64
	negativeEndingRuns := 0
	lossRuns := 0
	for i, r := range input.RMultiples {
		total += r
		equity += r
		if equity > peak {
			peak = equity
		}
		drawdown := peak - equity
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}
		if r < 0 {
			lossRuns++
			if lossRuns > result.MaxLossStreak {
				result.MaxLossStreak = lossRuns
			}
		} else {
			lossRuns = 0
		}
		if (i+1)%len(input.RMultiples) == 0 && equity < 0 {
			negativeEndingRuns++
		}
	}
	result.AverageR = math.Round((total/float64(len(input.RMultiples)))*100) / 100
	result.MaxDrawdownR = math.Round(maxDrawdown*100) / 100
	if input.SimulationCount > 0 {
		result.ProbabilityEndingNegative = float64(negativeEndingRuns) / float64(input.SimulationCount)
	}
	if result.AverageR < 0 || result.MaxDrawdownR >= 8 {
		result.Verdict = SimulationDisableStrategy
		result.Warnings = append(result.Warnings, "strategy risk exceeds limits")
	} else if result.MaxLossStreak >= 5 || result.MaxDrawdownR >= 4 {
		result.Verdict = SimulationReduceRisk
		result.Warnings = append(result.Warnings, "loss streak or drawdown requires risk reduction")
	} else {
		result.Verdict = SimulationAcceptable
	}
	return result
}

func normalizeAssetMap(input map[string]AssetState) map[string]AssetState {
	out := map[string]AssetState{}
	for symbol, state := range input {
		key := strings.ToUpper(strings.TrimSpace(symbol))
		if key == "" {
			continue
		}
		if state.Trend == "" {
			state.Trend = TrendUnknown
		}
		out[key] = state
	}
	return out
}

func missingAssets(assets map[string]AssetState, required []string) []string {
	missing := []string{}
	for _, symbol := range required {
		if _, ok := assets[symbol]; !ok {
			missing = append(missing, symbol)
		}
	}
	return missing
}

func boolScore(ok bool) int {
	if ok {
		return 1
	}
	return 0
}

func confidenceFromScore(score, maxScore, missing int) float64 {
	confidence := float64(score) / float64(maxScore)
	confidence -= float64(missing) * 0.03
	if confidence < 0.1 {
		confidence = 0.1
	}
	if confidence > 1 {
		confidence = 1
	}
	return math.Round(confidence*100) / 100
}

func snapshotReasonContext(MarketRegimeInput) string {
	return ""
}

func normalizeDirectionMap(input map[string]Direction) map[string]Direction {
	out := map[string]Direction{}
	for symbol, direction := range input {
		key := strings.ToUpper(strings.TrimSpace(symbol))
		if key == "" {
			continue
		}
		if direction == "" {
			direction = DirectionUnknown
		}
		out[key] = direction
	}
	return out
}

func directionMatches(want, got Direction) bool {
	return want == got || want == DirectionFlat && got != DirectionUnknown
}

func surpriseDrivenEvent(eventType string) bool {
	eventType = strings.ToUpper(strings.TrimSpace(eventType))
	return strings.Contains(eventType, "CPI") ||
		strings.Contains(eventType, "PPI") ||
		strings.Contains(eventType, "PAYROLL") ||
		strings.Contains(eventType, "UNEMPLOYMENT") ||
		strings.Contains(eventType, "EARNINGS") ||
		strings.Contains(eventType, "GDP") ||
		strings.Contains(eventType, "PMI") ||
		strings.Contains(eventType, "RATE_DECISION")
}

func calendarDirection(eventType string, surprise float64) CalendarDirection {
	eventType = strings.ToUpper(strings.TrimSpace(eventType))
	switch {
	case strings.Contains(eventType, "CPI") || strings.Contains(eventType, "PPI") || strings.Contains(eventType, "EARNINGS"):
		if surprise > 0 {
			return CalendarDirectionHawkishRates
		}
		if surprise < 0 {
			return CalendarDirectionDovishRates
		}
	case strings.Contains(eventType, "PAYROLL") || strings.Contains(eventType, "GDP") || strings.Contains(eventType, "PMI"):
		if surprise > 0 {
			return CalendarDirectionGrowthStrong
		}
		if surprise < 0 {
			return CalendarDirectionGrowthWeak
		}
	case strings.Contains(eventType, "UNEMPLOYMENT"):
		if surprise > 0 {
			return CalendarDirectionGrowthWeak
		}
		if surprise < 0 {
			return CalendarDirectionGrowthStrong
		}
	}
	return CalendarDirectionNeutral
}

func overlapsAny(left, right []string) bool {
	seen := map[string]struct{}{}
	for _, item := range left {
		seen[strings.ToUpper(strings.TrimSpace(item))] = struct{}{}
	}
	for _, item := range right {
		if _, ok := seen[strings.ToUpper(strings.TrimSpace(item))]; ok {
			return true
		}
	}
	return false
}

func confounderImpact(event ConfounderEvent, minutes float64, overlap bool) ConfounderImpact {
	if !overlap && event.Severity != SeverityCritical {
		return ConfounderInformationalOnly
	}
	if minutes <= 5 && (event.Severity == SeverityHigh || event.Severity == SeverityCritical || event.Type == ConfounderMegaCapEarnings) {
		return ConfounderBlocksTrade
	}
	if event.Type == ConfounderFedSpeaker || event.Type == ConfounderTreasuryAuction {
		return ConfounderRequiresManual
	}
	if event.Severity == SeverityMedium || event.Severity == SeverityHigh {
		return ConfounderReducesConfidence
	}
	return ConfounderInformationalOnly
}

func confounderReason(event ConfounderEvent, impact ConfounderImpact) string {
	if event.Reason != "" {
		return event.Reason
	}
	return fmt.Sprintf("%s confounder creates %s impact", event.Type, impact)
}

func defaultFloat(value, fallback float64) float64 {
	if value == 0 {
		return fallback
	}
	return value
}

func normalizedTime(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now().UTC()
	}
	return value.UTC()
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
