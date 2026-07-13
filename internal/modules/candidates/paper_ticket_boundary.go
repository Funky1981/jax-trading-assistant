package candidates

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

const (
	PaperTicketStatusBlocked              = "blocked"
	PaperTicketStatusApprovalNotReady     = "approval_not_ready"
	PaperTicketStatusApprovalRequired     = "approval_required"
	PaperTicketStatusPaperTicketReady     = "paper_ticket_ready"
	PaperTicketStatusPaperTicketCreated   = "paper_ticket_created"
	PaperTicketStatusPaperTicketCancelled = "paper_ticket_cancelled"
)

type PaperTicketRequest struct {
	Candidate            Candidate
	Eligibility          ApprovalEligibilityResult
	HumanApprovalGranted bool
	ApprovalDecisionRef  string
	SourceApprovalID     uuid.UUID
	CreatedAt            time.Time
}

type PaperTicketResult struct {
	CanCreateTicket             bool       `json:"can_create_ticket"`
	PaperTicketID               string     `json:"paper_ticket_id"`
	CandidateID                 uuid.UUID  `json:"candidate_id"`
	CreatedAt                   time.Time  `json:"created_at"`
	Status                      string     `json:"status"`
	SourceApprovalID            *uuid.UUID `json:"source_approval_id,omitempty"`
	ApprovalDecisionRef         string     `json:"approval_decision_ref,omitempty"`
	EntryPrice                  float64    `json:"entry_price"`
	StopLossPrice               float64    `json:"stop_loss_price"`
	TargetPrice                 float64    `json:"target_price"`
	PositionSize                float64    `json:"position_size"`
	MaxNormalLoss               float64    `json:"max_normal_loss"`
	MaxSlippageAdjustedLoss     float64    `json:"max_slippage_adjusted_loss"`
	RewardRiskRatio             float64    `json:"reward_risk_ratio"`
	PaperOnly                   bool       `json:"paper_only"`
	BrokerExecutionAllowed      bool       `json:"broker_execution_allowed"`
	ExecutionInstructionCreated bool       `json:"execution_instruction_created"`
	LiveTradingAllowed          bool       `json:"live_trading_allowed"`
	LeverageAllowed             bool       `json:"leverage_allowed"`
	RejectReasons               []string   `json:"reject_reasons,omitempty"`
}

func CreatePaperTicket(request PaperTicketRequest) PaperTicketResult {
	now := request.CreatedAt
	if now.IsZero() {
		now = time.Now().UTC()
	}
	candidate := request.Candidate
	result := PaperTicketResult{
		CanCreateTicket:             true,
		PaperTicketID:               fmt.Sprintf("pt_%s", candidate.ID.String()),
		CandidateID:                 candidate.ID,
		CreatedAt:                   now,
		Status:                      PaperTicketStatusPaperTicketReady,
		ApprovalDecisionRef:         request.ApprovalDecisionRef,
		EntryPrice:                  floatPtrValue(candidate.EntryPrice),
		StopLossPrice:               floatPtrValue(candidate.StopLoss),
		TargetPrice:                 floatPtrValue(candidate.TakeProfit),
		PositionSize:                floatPtrValue(candidate.PositionSize),
		MaxNormalLoss:               floatPtrValue(candidate.MaxNormalLoss),
		MaxSlippageAdjustedLoss:     floatPtrValue(candidate.MaxSlippageAdjustedLoss),
		RewardRiskRatio:             floatPtrValue(candidate.ExpectedRewardRiskRatio),
		PaperOnly:                   true,
		BrokerExecutionAllowed:      false,
		ExecutionInstructionCreated: false,
		LiveTradingAllowed:          false,
		LeverageAllowed:             false,
	}
	if request.SourceApprovalID != uuid.Nil {
		sourceApprovalID := request.SourceApprovalID
		result.SourceApprovalID = &sourceApprovalID
	}

	if !ValidateStructuralCompleteness(candidate).StructurallyComplete {
		blockPaperTicket(&result, PaperTicketStatusBlocked, "structural_fields_missing")
	}
	if !request.Eligibility.ApprovalEligible {
		blockPaperTicket(&result, PaperTicketStatusApprovalNotReady, "approval_eligibility_not_passed")
		result.RejectReasons = appendUnique(result.RejectReasons, request.Eligibility.RejectReasons...)
		result.RejectReasons = appendUnique(result.RejectReasons, request.Eligibility.WarningReasons...)
	}
	if !request.HumanApprovalGranted {
		blockPaperTicket(&result, PaperTicketStatusApprovalRequired, "human_approval_required")
	}
	if leverageRequestedAboveOne(candidate) {
		blockPaperTicket(&result, PaperTicketStatusBlocked, "leverage_requested_above_1")
	}
	if request.Eligibility.BrokerExecutionAllowed || metadataBool(candidate.Metadata, "brokerExecutionAllowed") {
		blockPaperTicket(&result, PaperTicketStatusBlocked, "broker_execution_allowed_too_early")
	}
	if request.Eligibility.ExecutionInstructionCreated || candidate.ExecutionInstructionID != nil || candidate.TradeID != nil || candidate.Status == StatusSubmitted || candidate.Status == StatusFilled {
		blockPaperTicket(&result, PaperTicketStatusBlocked, "execution_instruction_created_too_early")
	}
	if request.Eligibility.LiveTradingAllowed || metadataBool(candidate.Metadata, "liveTradingAllowed") {
		blockPaperTicket(&result, PaperTicketStatusBlocked, "live_trading_allowed_too_early")
	}
	return result
}

func blockPaperTicket(result *PaperTicketResult, status, reason string) {
	result.CanCreateTicket = false
	result.Status = status
	result.RejectReasons = appendUnique(result.RejectReasons, reason)
	result.BrokerExecutionAllowed = false
	result.ExecutionInstructionCreated = false
	result.LiveTradingAllowed = false
	result.LeverageAllowed = false
	result.PaperOnly = true
}

func floatPtrValue(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}
