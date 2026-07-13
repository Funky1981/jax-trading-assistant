package candidates

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

const NextPhaseApprovalReview = "approval_review"

type ApprovalEligibilityResult struct {
	CandidateID                 uuid.UUID `json:"candidateId"`
	EvaluatedAt                 time.Time `json:"evaluatedAt"`
	ApprovalEligible            bool      `json:"approvalEligible"`
	ApprovalStatus              string    `json:"approvalStatus"`
	RejectReasons               []string  `json:"rejectReasons,omitempty"`
	WarningReasons              []string  `json:"warningReasons,omitempty"`
	NextRequiredPhase           string    `json:"nextRequiredPhase"`
	BrokerExecutionAllowed      bool      `json:"brokerExecutionAllowed"`
	ExecutionInstructionCreated bool      `json:"executionInstructionCreated"`
	LiveTradingAllowed          bool      `json:"liveTradingAllowed"`
}

func EvaluateApprovalEligibility(candidate Candidate, evidence EvidenceScoreSummary, gate GateResult, risk RiskReviewResult, now time.Time) ApprovalEligibilityResult {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	result := ApprovalEligibilityResult{
		CandidateID:                 candidate.ID,
		EvaluatedAt:                 now,
		ApprovalEligible:            true,
		ApprovalStatus:              ApprovalStatusApprovalReviewReady,
		NextRequiredPhase:           NextPhaseApprovalReview,
		BrokerExecutionAllowed:      false,
		ExecutionInstructionCreated: false,
		LiveTradingAllowed:          false,
	}

	structural := ValidateStructuralCompleteness(candidate)
	if !structural.StructurallyComplete {
		blockApproval(&result, ApprovalStatusStructureIncomplete, NextPhaseCandidateRepair, "structural_fields_missing")
		result.RejectReasons = appendUnique(result.RejectReasons, structural.RejectReasons...)
		for _, missing := range structural.MissingFields {
			result.RejectReasons = appendUnique(result.RejectReasons, "missing_"+missing)
		}
	}

	applyApprovalEvidenceChecks(&result, evidence)
	applyApprovalGateChecks(&result, gate)
	applyApprovalRiskChecks(&result, risk)
	applyApprovalSafetyChecks(&result, candidate, evidence, gate, risk)

	if result.ApprovalEligible {
		result.ApprovalStatus = ApprovalStatusApprovalReviewReady
		result.NextRequiredPhase = NextPhaseApprovalReview
	}

	return result
}

func applyApprovalEvidenceChecks(result *ApprovalEligibilityResult, evidence EvidenceScoreSummary) {
	switch evidence.EvidenceStatus {
	case EvidenceStatusSufficient:
		if !evidence.EvidenceReady {
			blockApproval(result, ApprovalStatusEvidenceNotReady, NextPhaseEvidenceReview, "evidence_not_ready")
		}
	case "", EvidenceStatusMissing:
		blockApproval(result, ApprovalStatusEvidenceNotReady, NextPhaseEvidenceReview, "evidence_missing")
	case EvidenceStatusWeak:
		blockApproval(result, ApprovalStatusEvidenceNotReady, NextPhaseEvidenceReview, "evidence_weak")
	case EvidenceStatusMixed:
		blockApproval(result, ApprovalStatusEvidenceNotReady, NextPhaseEvidenceReview, "evidence_mixed")
	case EvidenceStatusStale:
		blockApproval(result, ApprovalStatusEvidenceNotReady, NextPhaseEvidenceReview, "evidence_stale")
	case EvidenceStatusBlocked:
		blockApproval(result, ApprovalStatusEvidenceNotReady, NextPhaseStop, "evidence_blocked")
	default:
		blockApproval(result, ApprovalStatusEvidenceNotReady, NextPhaseEvidenceReview, "evidence_status_unknown")
	}
}

func applyApprovalGateChecks(result *ApprovalEligibilityResult, gate GateResult) {
	if gate.GateReady && gate.GateStatus == GateStatusReadyForRiskReview {
		return
	}
	blockApproval(result, ApprovalStatusGateNotReady, firstNonEmpty(gate.NextRequiredPhase, NextPhaseRiskReview), "gate_not_ready")
	result.RejectReasons = appendUnique(result.RejectReasons, gate.RejectReasons...)
	result.WarningReasons = appendUnique(result.WarningReasons, gate.WarningReasons...)
}

func applyApprovalRiskChecks(result *ApprovalEligibilityResult, risk RiskReviewResult) {
	if risk.RiskReady && risk.RiskStatus == RiskStatusReadyForApprovalReview {
		if risk.MaxAllowedLoss > 0 && risk.MaxSlippageAdjustedLoss > risk.MaxAllowedLoss {
			blockApproval(result, ApprovalStatusRiskNotReady, NextPhaseRiskReview, "slippage_adjusted_loss_above_allowed")
		}
		return
	}
	blockApproval(result, ApprovalStatusRiskNotReady, firstNonEmpty(risk.NextRequiredPhase, NextPhaseRiskReview), "risk_not_ready")
	result.RejectReasons = appendUnique(result.RejectReasons, risk.RejectReasons...)
	result.WarningReasons = appendUnique(result.WarningReasons, risk.WarningReasons...)
}

func applyApprovalSafetyChecks(result *ApprovalEligibilityResult, candidate Candidate, evidence EvidenceScoreSummary, gate GateResult, risk RiskReviewResult) {
	if leverageRequestedAboveOne(candidate) {
		blockApproval(result, ApprovalStatusBlocked, NextPhaseStop, "leverage_requested_above_1")
	}
	if evidence.BrokerExecutionAllowed || gate.BrokerExecutionAllowed || risk.BrokerExecutionAllowed || metadataBool(candidate.Metadata, "brokerExecutionAllowed") {
		blockApproval(result, ApprovalStatusBlocked, NextPhaseStop, "broker_execution_allowed_too_early")
	}
	if evidence.ExecutionInstructionCreated || gate.ExecutionInstructionCreated || risk.ExecutionInstructionCreated ||
		candidate.ExecutionInstructionID != nil || candidate.TradeID != nil || candidate.Status == StatusSubmitted || candidate.Status == StatusFilled {
		blockApproval(result, ApprovalStatusBlocked, NextPhaseStop, "execution_instruction_created_too_early")
	}
	if evidence.ApprovalGranted || gate.ApprovalGranted || risk.ApprovalGranted || candidate.Status == StatusApproved ||
		strings.EqualFold(candidate.ApprovalStatus, "approved") ||
		(candidate.LatestApproval != nil && strings.EqualFold(candidate.LatestApproval.Decision, "approved")) {
		blockApproval(result, ApprovalStatusBlocked, NextPhaseStop, "approval_granted_too_early")
	}
}

func blockApproval(result *ApprovalEligibilityResult, status, nextPhase, reason string) {
	result.ApprovalEligible = false
	if result.ApprovalStatus == ApprovalStatusApprovalReviewReady || status == ApprovalStatusBlocked {
		result.ApprovalStatus = status
	}
	if strings.TrimSpace(nextPhase) != "" && (result.NextRequiredPhase == NextPhaseApprovalReview || nextPhase == NextPhaseStop) {
		result.NextRequiredPhase = nextPhase
	}
	result.RejectReasons = appendUnique(result.RejectReasons, reason)
}
