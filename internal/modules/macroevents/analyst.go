package macroevents

import (
	"context"
	"math"
	"strings"
	"time"
)

type AnalystDecision string

type AnalystHardVeto string

const (
	AnalystDecisionCandidateAllowed     AnalystDecision = "candidate_allowed"
	AnalystDecisionCandidateRejected    AnalystDecision = "candidate_rejected"
	AnalystDecisionWatchOnly            AnalystDecision = "watch_only"
	AnalystDecisionInsufficientEvidence AnalystDecision = "insufficient_evidence"
	AnalystDecisionManualReviewOnly     AnalystDecision = "manual_review_only"
)

const (
	AnalystHardVetoNoChartConfirmation   AnalystHardVeto = "no_chart_confirmation"
	AnalystHardVetoFundamentalConflicted AnalystHardVeto = "fundamental_verdict_conflicted"
	AnalystHardVetoPricedIn              AnalystHardVeto = "priced_in_verdict_priced_in"
	AnalystHardVetoPricedInUnclear       AnalystHardVeto = "priced_in_verdict_unclear"
	AnalystHardVetoMajorConfounder       AnalystHardVeto = "major_confounder_unresolved"
	AnalystHardVetoNoStop                AnalystHardVeto = "no_stop_level"
	AnalystHardVetoRewardRisk            AnalystHardVeto = "reward_risk_below_minimum"
	AnalystHardVetoETFNotAllowlisted     AnalystHardVeto = "etf_not_allowlisted"
	AnalystHardVetoMarketDataMissing     AnalystHardVeto = "market_data_missing"
	AnalystHardVetoSourceQualityTooLow   AnalystHardVeto = "source_quality_too_low"
	AnalystHardVetoLiveTradingRequested  AnalystHardVeto = "live_trading_requested"
)

type AnalystDecisionInput struct {
	MacroEventID         string
	Symbol               string
	Technical            TechnicalSnapshot
	Fundamental          FundamentalSnapshot
	PricedIn             PricedInScore
	RiskScore            float64
	ConfidenceScore      float64
	HasStopLevel         bool
	RewardRisk           float64
	Allowlisted          bool
	MarketDataMissing    bool
	LiveTradingRequested bool
	RiskGuardrail        string
	EntryStopTarget      string
	Confounders          []Confounder
	EvidenceBundleID     string
}

type AnalystDecisionRecord struct {
	ID                    string
	MacroEventID          string
	Symbol                string
	FundamentalSnapshotID string
	TechnicalSnapshotID   string
	EvidenceBundleID      string
	FundamentalScore      float64
	TechnicalScore        float64
	RiskScore             float64
	ConfidenceScore       float64
	CandidateScore        float64
	Decision              AnalystDecision
	HardVetoes            []AnalystHardVeto
	Reasons               []string
	CreatedAt             time.Time
}

type analystDecisionStore interface {
	SaveAnalystDecision(ctx context.Context, decision AnalystDecisionRecord) (AnalystDecisionRecord, error)
}

type AnalystScoringService struct {
	store analystDecisionStore
}

func NewAnalystScoringService(store analystDecisionStore) *AnalystScoringService {
	return &AnalystScoringService{store: store}
}

func (s *AnalystScoringService) EvaluateAndSave(ctx context.Context, input AnalystDecisionInput) (AnalystDecisionRecord, error) {
	record := ScoreAnalystDecision(input)
	if s.store == nil {
		return record, nil
	}
	return s.store.SaveAnalystDecision(ctx, record)
}

func ScoreAnalystDecision(input AnalystDecisionInput) AnalystDecisionRecord {
	symbol := strings.ToUpper(strings.TrimSpace(input.Symbol))
	record := AnalystDecisionRecord{
		MacroEventID:          strings.TrimSpace(input.MacroEventID),
		Symbol:                symbol,
		FundamentalSnapshotID: strings.TrimSpace(input.Fundamental.ID),
		TechnicalSnapshotID:   strings.TrimSpace(input.Technical.ID),
		EvidenceBundleID:      strings.TrimSpace(input.EvidenceBundleID),
		FundamentalScore:      clampAnalystScore(input.Fundamental.FundamentalScore),
		TechnicalScore:        clampAnalystScore(input.Technical.TechnicalScore),
		RiskScore:             clampAnalystScore(input.RiskScore),
		ConfidenceScore:       clampAnalystScore(input.ConfidenceScore),
		Reasons:               []string{},
		HardVetoes:            []AnalystHardVeto{},
		CreatedAt:             time.Now().UTC(),
	}

	if record.Symbol == "" {
		record.HardVetoes = append(record.HardVetoes, AnalystHardVetoETFNotAllowlisted)
		record.Reasons = append(record.Reasons, "symbol is required")
	}
	if !input.Allowlisted {
		record.HardVetoes = append(record.HardVetoes, AnalystHardVetoETFNotAllowlisted)
		record.Reasons = append(record.Reasons, "ETF is not allowlisted")
	}
	if input.LiveTradingRequested {
		record.HardVetoes = append(record.HardVetoes, AnalystHardVetoLiveTradingRequested)
		record.Reasons = append(record.Reasons, "live trading is not allowed in phase 1")
	}
	if input.MarketDataMissing || input.Technical.Verdict == TechnicalVerdictInsufficientData || input.Technical.Verdict == TechnicalVerdictNoConfirmation {
		record.HardVetoes = append(record.HardVetoes, AnalystHardVetoMarketDataMissing)
		record.Reasons = append(record.Reasons, "market data is missing or insufficient")
	}
	if input.Fundamental.Verdict == FundamentalVerdictConflicted {
		record.HardVetoes = append(record.HardVetoes, AnalystHardVetoFundamentalConflicted)
		record.Reasons = append(record.Reasons, "fundamental verdict is conflicted")
	}
	if input.PricedIn.Verdict == PricedInVerdictPricedIn {
		record.HardVetoes = append(record.HardVetoes, AnalystHardVetoPricedIn)
		record.Reasons = append(record.Reasons, "priced-in verdict is priced_in")
	}
	if input.PricedIn.Verdict == PricedInVerdictUnclear {
		record.HardVetoes = append(record.HardVetoes, AnalystHardVetoPricedInUnclear)
		record.Reasons = append(record.Reasons, "priced-in verdict is unclear")
	}
	if input.PricedIn.BlocksCandidate {
		record.HardVetoes = append(record.HardVetoes, AnalystHardVetoPricedIn)
	}
	if !input.HasStopLevel {
		record.HardVetoes = append(record.HardVetoes, AnalystHardVetoNoStop)
		record.Reasons = append(record.Reasons, "stop level is missing")
	}
	if input.RewardRisk < 1.5 {
		record.HardVetoes = append(record.HardVetoes, AnalystHardVetoRewardRisk)
		record.Reasons = append(record.Reasons, "reward:risk is below minimum")
	}
	if strings.TrimSpace(input.RiskGuardrail) == "" {
		record.HardVetoes = append(record.HardVetoes, AnalystHardVetoSourceQualityTooLow)
		record.Reasons = append(record.Reasons, "risk guardrail is missing")
	}
	if input.Fundamental.Verdict == FundamentalVerdictInsufficientData || containsStringFold(input.Fundamental.MissingEvidence, "source quality") {
		record.HardVetoes = append(record.HardVetoes, AnalystHardVetoSourceQualityTooLow)
		record.Reasons = append(record.Reasons, "source quality is too low")
	}
	for _, confounder := range input.Confounders {
		if confounder.BlocksCandidate {
			record.HardVetoes = append(record.HardVetoes, AnalystHardVetoMajorConfounder)
			record.Reasons = append(record.Reasons, "major confounder is unresolved")
			break
		}
	}
	if input.EntryStopTarget == "" {
		record.HardVetoes = append(record.HardVetoes, AnalystHardVetoNoStop)
	}

	record.CandidateScore = combineAnalystScores(record.FundamentalScore, record.TechnicalScore, record.RiskScore, record.ConfidenceScore)
	record.CandidateScore = math.Round(record.CandidateScore*100) / 100
	record.HardVetoes = uniqueAnalystHardVetoes(record.HardVetoes)

	record.Decision = decideAnalystOutcome(record, input)
	if len(record.Reasons) == 0 {
		record.Reasons = append(record.Reasons, "analysis complete")
	}
	return record
}

func decideAnalystOutcome(record AnalystDecisionRecord, input AnalystDecisionInput) AnalystDecision {
	if onlyEvidenceMissingVetoes(record.HardVetoes) {
		return AnalystDecisionInsufficientEvidence
	}
	if len(record.HardVetoes) > 0 {
		if containsAnalystHardVeto(record.HardVetoes, AnalystHardVetoLiveTradingRequested) || containsAnalystHardVeto(record.HardVetoes, AnalystHardVetoMarketDataMissing) || containsAnalystHardVeto(record.HardVetoes, AnalystHardVetoFundamentalConflicted) || containsAnalystHardVeto(record.HardVetoes, AnalystHardVetoPricedIn) || containsAnalystHardVeto(record.HardVetoes, AnalystHardVetoPricedInUnclear) || containsAnalystHardVeto(record.HardVetoes, AnalystHardVetoNoStop) || containsAnalystHardVeto(record.HardVetoes, AnalystHardVetoRewardRisk) || containsAnalystHardVeto(record.HardVetoes, AnalystHardVetoSourceQualityTooLow) || containsAnalystHardVeto(record.HardVetoes, AnalystHardVetoETFNotAllowlisted) || containsAnalystHardVeto(record.HardVetoes, AnalystHardVetoMajorConfounder) {
			return AnalystDecisionCandidateRejected
		}
	}

	switch {
	case record.CandidateScore >= 85:
		return AnalystDecisionCandidateAllowed
	case record.CandidateScore >= 75:
		return AnalystDecisionCandidateAllowed
	case record.CandidateScore >= 65:
		return AnalystDecisionManualReviewOnly
	case record.CandidateScore >= 50:
		return AnalystDecisionWatchOnly
	case record.CandidateScore > 0:
		return AnalystDecisionCandidateRejected
	default:
		return AnalystDecisionInsufficientEvidence
	}
}

func combineAnalystScores(fundamental, technical, risk, confidence float64) float64 {
	return (fundamental * 0.40) + (technical * 0.40) + (risk * 0.15) + (confidence * 0.05)
}

func clampAnalystScore(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

func uniqueAnalystHardVetoes(values []AnalystHardVeto) []AnalystHardVeto {
	seen := map[AnalystHardVeto]bool{}
	out := make([]AnalystHardVeto, 0, len(values))
	for _, value := range values {
		if seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func containsAnalystHardVeto(values []AnalystHardVeto, want AnalystHardVeto) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func onlyEvidenceMissingVetoes(values []AnalystHardVeto) bool {
	if len(values) == 0 {
		return false
	}
	for _, value := range values {
		switch value {
		case AnalystHardVetoMarketDataMissing, AnalystHardVetoSourceQualityTooLow, AnalystHardVetoNoChartConfirmation:
			continue
		default:
			return false
		}
	}
	return true
}

func containsStringFold(values []string, want string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(want)) {
			return true
		}
	}
	return false
}
