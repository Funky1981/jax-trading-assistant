package eventdecisions

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"

	"jax-trading-assistant/internal/modules/candidates"
	"jax-trading-assistant/internal/modules/instruments"
)

type Evaluator struct {
	Ruleset Ruleset
	Catalog *instruments.Catalog
}

func Eligible(event Event) (bool, string) {
	if event.Status == "rejected" {
		return false, "rejected_event"
	}
	if event.Status == "ignored" {
		return false, "ignored_or_deduplicated_event"
	}
	if !event.ProvenanceAvailable {
		return false, "missing_provenance"
	}
	if event.IsSynthetic {
		return false, "synthetic_or_fixture_event"
	}
	if event.NormalizedEventID == nil {
		return false, "missing_normalized_event"
	}
	if len(event.SourceURLs) == 0 || strings.TrimSpace(event.SourceURLs[0]) == "" {
		return false, "missing_source_url"
	}
	if event.PublicationAt.IsZero() || event.ReceiptAt.IsZero() {
		return false, "missing_required_event_timestamp"
	}
	return true, ""
}

func (e Evaluator) Evaluate(event Event) (Result, error) {
	if err := e.Ruleset.Validate(); err != nil {
		return Result{}, err
	}
	if ok, reason := Eligible(event); !ok {
		return Result{}, fmt.Errorf("event is not eligible: %s", reason)
	}
	assets := normalizedAssets(event.AffectedAssets)
	result := Result{
		Decision:            DecisionNoTrade,
		AffectedAssets:      assets,
		UnknownAssets:       len(assets) == 0,
		Reasons:             []string{},
		BlockingReasons:     []string{},
		MissingEvidence:     []string{},
		TrustGateState:      "not_evaluated",
		RiskReviewState:     "not_evaluated",
		EvidenceScore:       bounded(event.Confidence),
		EvidenceScoreSource: "world_monitor_confidence",
		AssetMappingProvenance: map[string]any{
			"mappingReason":     event.MappingReason,
			"mappingMethods":    append([]string{}, event.MappingMethods...),
			"normalizedEventId": event.NormalizedEventID.String(),
		},
	}
	if result.UnknownAssets {
		result.MissingEvidence = append(result.MissingEvidence, "truthful_asset_mapping")
		result.BlockingReasons = append(result.BlockingReasons, "unknown_assets_prevent_candidate")
	}

	material := e.isMaterial(event.Severity)
	if !material {
		result.Reasons = append(result.Reasons, "event_severity_is_not_financially_material_under_ruleset")
		result.BlockingReasons = append(result.BlockingReasons, "materiality_threshold_not_met")
		return normalizeResult(result), nil
	}
	if event.Confidence < e.Ruleset.WatchConfidenceMinimum {
		result.Reasons = append(result.Reasons, "persisted_confidence_is_below_watch_threshold")
		result.BlockingReasons = append(result.BlockingReasons, "watch_confidence_threshold_not_met")
		return normalizeResult(result), nil
	}

	if event.Candidate == nil {
		result.Decision = DecisionWatch
		result.Reasons = append(result.Reasons, "material_event_requires_continued_observation")
		result.MissingEvidence = append(result.MissingEvidence, "complete_structured_trade_candidate", "candidate_evidence_score", "trust_gate_review", "risk_review")
		if result.UnknownAssets {
			result.Reasons = append(result.Reasons, "material_catalyst_has_no_truthful_persisted_asset_mapping")
		} else {
			result.Reasons = append(result.Reasons, "persisted_asset_mapping_exists_but_no_complete_candidate_contract_exists")
		}
		return normalizeResult(result), nil
	}

	candidate := event.Candidate
	result.TrustGateState = emptyState(candidate.GateStatus)
	result.RiskReviewState = emptyState(candidate.RiskStatus)
	if event.CandidateEvidenceScore != nil {
		result.EvidenceScore = bounded(event.CandidateEvidenceScore.OverallEvidenceScore)
		result.EvidenceScoreSource = "candidate_evidence_scores"
	}

	if result.UnknownAssets {
		result.Decision = DecisionWatch
		result.Reasons = append(result.Reasons, "material_event_is_watched_while_asset_mapping_remains_unknown")
		return normalizeResult(result), nil
	}

	unsafeReasons := e.candidateSafetyReasons(event)
	if len(unsafeReasons) > 0 {
		result.BlockingReasons = append(result.BlockingReasons, unsafeReasons...)
		result.Reasons = append(result.Reasons, "existing_candidate_does_not_satisfy_safe_product_boundaries")
		return normalizeResult(result), nil
	}
	if candidate.Status == candidates.StatusBlocked || candidate.Status == candidates.StatusExpired || candidate.Status == candidates.StatusRejected {
		result.BlockingReasons = append(result.BlockingReasons, "candidate_lifecycle_not_active")
	}

	structural := candidates.ValidateStructuralCompleteness(*candidate)
	if !structural.StructurallyComplete {
		result.MissingEvidence = append(result.MissingEvidence, structural.MissingFields...)
		result.BlockingReasons = append(result.BlockingReasons, structural.RejectReasons...)
	}
	if event.CandidateEvidenceScore == nil {
		result.MissingEvidence = append(result.MissingEvidence, "candidate_evidence_score")
		result.BlockingReasons = append(result.BlockingReasons, "candidate_evidence_missing")
	} else {
		score := event.CandidateEvidenceScore
		if score.OverallEvidenceScore < e.Ruleset.CandidateEvidenceMinimum || !score.EvidenceReady || !score.EvidenceGateReady || score.EvidenceStatus != candidates.EvidenceStatusSufficient {
			result.BlockingReasons = append(result.BlockingReasons, "candidate_evidence_threshold_or_readiness_not_met")
		}
		if score.ContradictoryItemCount > 0 {
			result.BlockingReasons = append(result.BlockingReasons, "contradictory_candidate_evidence")
		}
		if score.StaleItemCount > 0 {
			result.BlockingReasons = append(result.BlockingReasons, "stale_candidate_evidence")
		}
	}
	if candidate.GateStatus != candidates.GateStatusReadyForRiskReview {
		result.MissingEvidence = append(result.MissingEvidence, "passed_trust_gate")
		result.BlockingReasons = append(result.BlockingReasons, "trust_gate_not_ready")
	}
	if candidate.RiskStatus != string(candidates.RiskStatusReadyForApprovalReview) {
		result.MissingEvidence = append(result.MissingEvidence, "passed_risk_review")
		result.BlockingReasons = append(result.BlockingReasons, "risk_review_not_ready")
	}
	if candidate.TakeProfit == nil || *candidate.TakeProfit <= 0 {
		result.MissingEvidence = append(result.MissingEvidence, "target_price")
		result.BlockingReasons = append(result.BlockingReasons, "candidate_target_missing")
	}

	if len(result.BlockingReasons) == 0 {
		result.Decision = DecisionCandidate
		id := candidate.ID
		result.CandidateID = &id
		result.Reasons = append(result.Reasons,
			"existing_structured_candidate_is_complete",
			"persisted_evidence_threshold_is_met",
			"trust_gate_and_risk_review_are_ready",
			"paper_only_product_and_leverage_boundaries_are_preserved",
		)
		return normalizeResult(result), nil
	}

	if candidate.Status == candidates.StatusBlocked || candidate.Status == candidates.StatusExpired || candidate.Status == candidates.StatusRejected {
		result.Decision = DecisionNoTrade
		result.Reasons = append(result.Reasons, "existing_candidate_is_blocked_expired_or_rejected")
	} else {
		result.Decision = DecisionWatch
		result.Reasons = append(result.Reasons, "material_event_cannot_become_a_candidate_until_persisted_contract_gaps_are_resolved")
	}
	return normalizeResult(result), nil
}

func (e Evaluator) isMaterial(severity string) bool {
	for _, allowed := range e.Ruleset.MaterialSeverities {
		if strings.EqualFold(strings.TrimSpace(severity), strings.TrimSpace(allowed)) {
			return true
		}
	}
	return false
}

func (e Evaluator) candidateSafetyReasons(event Event) []string {
	candidate := event.Candidate
	reasons := []string{}
	if !strings.EqualFold(candidate.InstrumentType, e.Ruleset.AllowedCandidateInstrumentType) {
		reasons = append(reasons, "unsafe_or_unknown_candidate_product")
	}
	if !containsAsset(event.AffectedAssets, candidate.Symbol) {
		reasons = append(reasons, "candidate_symbol_not_supported_by_persisted_asset_mapping")
	}
	if e.Catalog == nil {
		reasons = append(reasons, "instrument_catalog_unavailable")
	} else if evaluation := e.Catalog.Evaluate(candidate.Symbol, "paper"); !evaluation.Allowed {
		reasons = append(reasons, "instrument_policy_"+evaluation.ReasonCode)
	}
	if requestedLeverage(candidate.Metadata) > e.Ruleset.MaximumLeverage {
		reasons = append(reasons, "leverage_above_1x")
	}
	if candidate.ExecutionInstructionID != nil || candidate.TradeID != nil {
		reasons = append(reasons, "candidate_has_execution_side_linkage")
	}
	return reasons
}

func requestedLeverage(raw *json.RawMessage) float64 {
	if raw == nil || len(*raw) == 0 {
		return 1
	}
	var value any
	if json.Unmarshal(*raw, &value) != nil {
		return math.Inf(1)
	}
	var walk func(any) float64
	walk = func(current any) float64 {
		switch typed := current.(type) {
		case map[string]any:
			max := 1.0
			for key, nested := range typed {
				normalized := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(key), "_", ""))
				if normalized == "leverage" || normalized == "requestedleverage" || normalized == "maxleverage" {
					if number, ok := nested.(float64); ok && number > max {
						max = number
					}
				}
				if nestedMax := walk(nested); nestedMax > max {
					max = nestedMax
				}
			}
			return max
		case []any:
			max := 1.0
			for _, nested := range typed {
				if nestedMax := walk(nested); nestedMax > max {
					max = nestedMax
				}
			}
			return max
		default:
			return 1
		}
	}
	return walk(value)
}

func normalizedAssets(raw []string) []string {
	seen := map[string]bool{}
	assets := []string{}
	for _, item := range raw {
		asset := strings.ToUpper(strings.TrimSpace(item))
		if asset != "" && !seen[asset] {
			seen[asset] = true
			assets = append(assets, asset)
		}
	}
	sort.Strings(assets)
	return assets
}

func containsAsset(assets []string, symbol string) bool {
	for _, asset := range assets {
		if strings.EqualFold(strings.TrimSpace(asset), strings.TrimSpace(symbol)) {
			return true
		}
	}
	return false
}

func bounded(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func emptyState(value string) string {
	if strings.TrimSpace(value) == "" {
		return "not_evaluated"
	}
	return value
}

func normalizeResult(result Result) Result {
	result.Reasons = uniqueSorted(result.Reasons)
	result.BlockingReasons = uniqueSorted(result.BlockingReasons)
	result.MissingEvidence = uniqueSorted(result.MissingEvidence)
	result.AffectedAssets = normalizedAssets(result.AffectedAssets)
	if result.Decision != DecisionCandidate {
		result.CandidateID = nil
	}
	return result
}

func uniqueSorted(values []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	sort.Strings(out)
	return out
}
