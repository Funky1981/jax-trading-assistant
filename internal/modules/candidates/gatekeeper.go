package candidates

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	NextPhaseRiskReview      = "risk_review"
	NextPhaseEvidenceReview  = "evidence_review"
	NextPhaseCandidateRepair = "candidate_repair"
	NextPhaseStop            = "stop"
)

type GateResult struct {
	CandidateID                 uuid.UUID `json:"candidateId"`
	EvaluatedAt                 time.Time `json:"evaluatedAt"`
	GateStatus                  string    `json:"gateStatus"`
	GateReady                   bool      `json:"gateReady"`
	HardReject                  bool      `json:"hardReject"`
	RejectReasons               []string  `json:"rejectReasons,omitempty"`
	WarningReasons              []string  `json:"warningReasons,omitempty"`
	NextRequiredPhase           string    `json:"nextRequiredPhase"`
	BrokerExecutionAllowed      bool      `json:"brokerExecutionAllowed"`
	ExecutionInstructionCreated bool      `json:"executionInstructionCreated"`
	ApprovalGranted             bool      `json:"approvalGranted"`
}

func EvaluateCandidateGate(candidate Candidate, evidence EvidenceScoreSummary, now time.Time) GateResult {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	result := GateResult{
		CandidateID:                 candidate.ID,
		EvaluatedAt:                 now,
		GateStatus:                  GateStatusReadyForRiskReview,
		GateReady:                   true,
		NextRequiredPhase:           NextPhaseRiskReview,
		RejectReasons:               append([]string{}, candidate.RejectReasons...),
		WarningReasons:              nil,
		ApprovalGranted:             false,
		BrokerExecutionAllowed:      false,
		ExecutionInstructionCreated: false,
	}

	structural := ValidateStructuralCompleteness(candidate)
	if !structural.StructurallyComplete {
		result.GateStatus = GateStatusIncomplete
		result.GateReady = false
		result.HardReject = true
		result.NextRequiredPhase = NextPhaseCandidateRepair
		result.RejectReasons = appendUnique(result.RejectReasons, structural.RejectReasons...)
		for _, missing := range structural.MissingFields {
			result.RejectReasons = appendUnique(result.RejectReasons, "missing_"+missing)
		}
	}
	if containsString(structural.RejectReasons, "contradictory_evidence_present") {
		result.GateStatus = GateStatusBlocked
		result.GateReady = false
		result.HardReject = true
		result.NextRequiredPhase = NextPhaseStop
		result.RejectReasons = appendUnique(result.RejectReasons, structural.RejectReasons...)
	}

	applyEvidenceGate(&result, evidence)
	applySafetyGate(&result, candidate, evidence)

	if result.HardReject {
		result.GateReady = false
		if result.GateStatus == GateStatusReadyForRiskReview {
			result.GateStatus = GateStatusBlocked
		}
		if result.NextRequiredPhase == NextPhaseRiskReview && result.GateStatus == GateStatusBlocked {
			result.NextRequiredPhase = NextPhaseStop
		}
		return result
	}

	if result.GateStatus != GateStatusReadyForRiskReview {
		result.GateReady = false
		return result
	}

	if normalizedRiskStatus(candidate.RiskStatus) == RiskStatusPending {
		result.WarningReasons = appendUnique(result.WarningReasons, "risk_review_pending")
	}
	if strings.TrimSpace(candidate.ApprovalStatus) == "" || candidate.ApprovalStatus == ApprovalStatusNotReady {
		result.WarningReasons = appendUnique(result.WarningReasons, "approval_not_started")
	}

	return result
}

func applyEvidenceGate(result *GateResult, evidence EvidenceScoreSummary) {
	switch evidence.EvidenceStatus {
	case "", EvidenceStatusMissing:
		blockWithoutHardReject(result, GateStatusEvidenceMissing, NextPhaseEvidenceReview, "evidence_missing")
	case EvidenceStatusWeak:
		blockWithoutHardReject(result, GateStatusEvidenceWeak, NextPhaseEvidenceReview, "evidence_weak")
	case EvidenceStatusMixed:
		blockWithoutHardReject(result, GateStatusEvidenceMixed, NextPhaseEvidenceReview, "evidence_mixed")
	case EvidenceStatusStale:
		hardReject(result, GateStatusEvidenceStale, NextPhaseEvidenceReview, "evidence_stale")
	case EvidenceStatusBlocked:
		hardReject(result, GateStatusBlocked, NextPhaseStop, "evidence_blocked")
	case EvidenceStatusSufficient:
		if !evidence.EvidenceReady || !evidence.EvidenceGateReady {
			blockWithoutHardReject(result, GateStatusEvidenceWeak, NextPhaseEvidenceReview, "evidence_not_ready")
		}
	default:
		blockWithoutHardReject(result, GateStatusEvidenceMissing, NextPhaseEvidenceReview, "evidence_status_unknown")
	}
	if evidence.ContradictoryItemCount > 0 && evidence.SupportingItemCount == 0 {
		hardReject(result, GateStatusBlocked, NextPhaseStop, "only_contradictory_evidence")
	}
}

func applySafetyGate(result *GateResult, candidate Candidate, evidence EvidenceScoreSummary) {
	if leverageRequestedAboveOne(candidate) {
		hardReject(result, GateStatusBlocked, NextPhaseStop, "leverage_requested_above_1")
	}
	if evidence.BrokerExecutionAllowed || metadataBool(candidate.Metadata, "brokerExecutionAllowed") {
		hardReject(result, GateStatusBlocked, NextPhaseStop, "broker_execution_allowed_too_early")
	}
	if evidence.ExecutionInstructionCreated || candidate.ExecutionInstructionID != nil || candidate.TradeID != nil ||
		candidate.Status == StatusSubmitted || candidate.Status == StatusFilled {
		hardReject(result, GateStatusBlocked, NextPhaseStop, "execution_instruction_created_too_early")
	}
	if evidence.ApprovalGranted || candidate.Status == StatusApproved || strings.EqualFold(candidate.ApprovalStatus, "approved") ||
		(candidate.LatestApproval != nil && strings.EqualFold(candidate.LatestApproval.Decision, "approved")) {
		hardReject(result, GateStatusBlocked, NextPhaseStop, "approval_granted_too_early")
	}
}

func blockWithoutHardReject(result *GateResult, status, nextPhase, reason string) {
	if result.HardReject {
		return
	}
	result.GateStatus = status
	result.GateReady = false
	result.NextRequiredPhase = nextPhase
	result.WarningReasons = appendUnique(result.WarningReasons, reason)
}

func hardReject(result *GateResult, status, nextPhase, reason string) {
	result.GateStatus = status
	result.GateReady = false
	result.HardReject = true
	result.NextRequiredPhase = nextPhase
	result.RejectReasons = appendUnique(result.RejectReasons, reason)
}

func appendUnique(values []string, additions ...string) []string {
	for _, addition := range additions {
		addition = strings.TrimSpace(addition)
		if addition == "" || containsString(values, addition) {
			continue
		}
		values = append(values, addition)
	}
	return values
}

func leverageRequestedAboveOne(candidate Candidate) bool {
	if candidate.Metadata == nil || len(*candidate.Metadata) == 0 {
		return false
	}
	var payload any
	if err := json.Unmarshal(*candidate.Metadata, &payload); err != nil {
		return false
	}
	return jsonNumberAboveOne(payload, map[string]bool{
		"leverage":           true,
		"requestedleverage":  true,
		"requested_leverage": true,
		"maxleverage":        true,
		"max_leverage":       true,
	})
}

func metadataBool(raw *json.RawMessage, key string) bool {
	if raw == nil || len(*raw) == 0 {
		return false
	}
	var payload any
	if err := json.Unmarshal(*raw, &payload); err != nil {
		return false
	}
	return jsonBool(payload, strings.ToLower(key))
}

func jsonNumberAboveOne(value any, keys map[string]bool) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, nested := range typed {
			normalized := strings.ToLower(strings.TrimSpace(key))
			if keys[normalized] {
				if number, ok := nested.(float64); ok && number > 1.0 {
					return true
				}
			}
			if jsonNumberAboveOne(nested, keys) {
				return true
			}
		}
	case []any:
		for _, nested := range typed {
			if jsonNumberAboveOne(nested, keys) {
				return true
			}
		}
	}
	return false
}

func jsonBool(value any, wantKey string) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, nested := range typed {
			if strings.ToLower(strings.TrimSpace(key)) == wantKey {
				if flag, ok := nested.(bool); ok {
					return flag
				}
			}
			if jsonBool(nested, wantKey) {
				return true
			}
		}
	case []any:
		for _, nested := range typed {
			if jsonBool(nested, wantKey) {
				return true
			}
		}
	}
	return false
}
