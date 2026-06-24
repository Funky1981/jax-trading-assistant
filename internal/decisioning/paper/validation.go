package paper

import (
	"time"

	"jax-trading-assistant/internal/decisioning/core"
	"jax-trading-assistant/internal/decisioning/risk"
)

const minimumRiskReward = 2.0

type ValidationResult struct {
	IsValid               bool     `json:"is_valid"`
	ValidationErrors      []string `json:"validation_errors"`
	ValidationWarnings    []string `json:"validation_warnings"`
	CanCreateTicket       bool     `json:"can_create_ticket"`
	CanApproveForPaper    bool     `json:"can_approve_for_paper"`
	RequiredHumanApproval bool     `json:"required_human_approval"`
	PaperOnly             bool     `json:"paper_only"`
	LiveTradingBlocked    bool     `json:"live_trading_blocked"`
	ForbiddenActions      []string `json:"forbidden_actions"`
	RequiredRemediation   []string `json:"required_remediation"`
}

func ValidateTicketRequest(request TicketRequest) ValidationResult {
	result := baseValidationResult(request.RiskAssessment.ForbiddenActions, request.SourceDecision.ForbiddenActions)
	fail := func(message string) {
		result.IsValid = false
		result.ValidationErrors = append(result.ValidationErrors, message)
		result.RequiredRemediation = append(result.RequiredRemediation, message)
	}

	if request.SourceDecision.Decision != core.DecisionTradeCandidate {
		fail("only TRADE_CANDIDATE can create a paper ticket")
	}
	if request.SourceDecision.Decision == core.DecisionNoTrade {
		fail("NO_TRADE cannot create a paper ticket")
	}
	if request.SourceDecision.Decision == core.DecisionWatch {
		fail("WATCH cannot create a paper ticket")
	}
	if request.SourceDecision.Decision == core.DecisionSetupForming {
		fail("SETUP_FORMING cannot create a paper ticket")
	}
	if request.SourceDecision.Decision == core.DecisionRejectedByRisk || request.RiskAssessment.FinalDecision == core.DecisionRejectedByRisk {
		fail("REJECTED_BY_RISK cannot create a paper ticket")
	}
	if request.RiskAssessment.RiskDecision != risk.RiskDecisionPass {
		fail("risk veto must pass before paper ticket creation")
	}
	if !request.RiskAssessment.RequiresHumanApproval || !request.ExplicitHumanApprovalRequired {
		fail("human approval must be required")
	}
	if !request.RiskAssessment.PaperOnly {
		fail("ticket must be paper-only")
	}
	if !request.RiskAssessment.LiveTradingBlocked {
		fail("live trading must be blocked")
	}
	if !request.ResearchEvidenceSummary.IsDefined() {
		result.ValidationWarnings = append(result.ValidationWarnings, "research evidence summary is missing")
		result.RequiredRemediation = append(result.RequiredRemediation, "attach research evidence summary before review")
	}
	if request.Asset == "" {
		fail("asset is required")
	}
	if request.SetupFamily == "" {
		fail("setup_family is required")
	}
	if len(request.InvalidationConditions) == 0 {
		fail("invalidation_conditions are required")
	}
	if request.ProposedStop <= 0 {
		fail("proposed_stop is required")
	}
	if request.ProposedTarget <= 0 {
		fail("proposed_target is required")
	}
	if !request.ProposedEntryZone.IsDefined() && request.EntryUnavailableReason == "" {
		fail("proposed_entry_zone or entry_unavailable_reason is required")
	}
	if request.RiskReward <= 0 {
		fail("risk_reward is required")
	}
	if request.RiskReward > 0 && request.RiskReward < minimumRiskReward {
		fail("risk_reward must be >= 2:1")
	}
	if request.MaxPaperPositionSize <= 0 && request.RiskAssessment.MaxPositionSize <= 0 {
		result.ValidationWarnings = append(result.ValidationWarnings, "max_paper_position_size is not calculable")
		result.RequiredRemediation = append(result.RequiredRemediation, "provide max paper position size when calculable")
	}
	if request.ExpiresAt.IsZero() {
		fail("expires_at is required")
	}
	if !request.CreatedAt.IsZero() && !request.ExpiresAt.IsZero() && !request.ExpiresAt.After(request.CreatedAt) {
		fail("expires_at must be after created_at")
	}
	if containsAny(request.RiskAssessment.AllowedActions, []string{core.ActionExecuteTrade, core.ActionCreateLiveOrder, core.ActionAutoApprove}) ||
		containsAny(request.SourceDecision.AllowedActions, []string{core.ActionExecuteTrade, core.ActionCreateLiveOrder, core.ActionAutoApprove}) {
		fail("allowed actions must not include live order, execution, or auto approval")
	}

	result.CanCreateTicket = result.IsValid
	return result
}

func ValidateApprovalRequest(request ApprovalRequest) ValidationResult {
	result := baseValidationResult(request.Ticket.ForbiddenActions)
	fail := func(message string) {
		result.IsValid = false
		result.ValidationErrors = append(result.ValidationErrors, message)
		result.RequiredRemediation = append(result.RequiredRemediation, message)
	}

	if request.Ticket.PaperTicketID == "" {
		fail("paper_ticket_id is required")
	}
	if request.Ticket.HumanApprovalStatus != ApprovalPendingReview {
		fail("only PENDING_REVIEW tickets can be approved")
	}
	if isTerminalStatus(request.Ticket.HumanApprovalStatus) {
		fail("terminal tickets cannot be approved")
	}
	if request.Ticket.HumanApprovalStatus == ApprovalDeferred {
		fail("DEFERRED tickets require a new review before approval")
	}
	if request.Ticket.HumanApprovalStatus == ApprovalRejectedByUser {
		fail("REJECTED_BY_USER tickets cannot be approved")
	}
	if request.Ticket.HumanApprovalStatus == ApprovalExpired || (!request.Now.IsZero() && request.Now.After(request.Ticket.ExpiresAt)) {
		fail("expired tickets cannot be approved")
	}
	if !request.ExplicitHumanApproval || request.AutomaticApproval {
		fail("approval must be explicit and human")
	}
	if !request.Ticket.PaperOnly {
		fail("ticket must be paper-only")
	}
	if !request.Ticket.LiveTradingBlocked {
		fail("live trading must be blocked")
	}
	if request.ApprovedBy == "" {
		fail("approved_by is required")
	}

	result.CanApproveForPaper = result.IsValid
	return result
}

func baseValidationResult(forbiddenSources ...[]string) ValidationResult {
	forbidden := mandatoryForbiddenActions(forbiddenSources...)
	return ValidationResult{
		IsValid:               true,
		RequiredHumanApproval: true,
		PaperOnly:             true,
		LiveTradingBlocked:    true,
		ForbiddenActions:      forbidden,
	}
}

func mandatoryForbiddenActions(sources ...[]string) []string {
	return appendUnique([]string{}, append(sources, []string{
		core.ActionExecuteTrade,
		core.ActionCreateLiveOrder,
		core.ActionAutoApprove,
	})...)
}

func appendUnique(base []string, sources ...[]string) []string {
	seen := map[string]bool{}
	for _, item := range base {
		seen[item] = true
	}
	for _, source := range sources {
		for _, item := range source {
			if item == "" || seen[item] {
				continue
			}
			base = append(base, item)
			seen[item] = true
		}
	}
	return base
}

func containsAny(values []string, needles []string) bool {
	seen := map[string]bool{}
	for _, value := range values {
		seen[value] = true
	}
	for _, needle := range needles {
		if seen[needle] {
			return true
		}
	}
	return false
}

func ticketExpired(ticket PaperTicket, now time.Time) bool {
	return !now.IsZero() && now.After(ticket.ExpiresAt)
}
