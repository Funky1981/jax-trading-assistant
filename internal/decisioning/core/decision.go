package core

const (
	BrainDecisionCore = "DECISION_CORE"

	ActionStoreEvent      = "store_event"
	ActionMonitor         = "monitor"
	ActionReviewLater     = "review_later"
	ActionPrepareResearch = "prepare_research"
	ActionPreparePaper    = "prepare_paper_candidate"
	ActionExecuteTrade    = "execute_trade"
	ActionCreateLiveOrder = "create_live_order"
	ActionAutoApprove     = "auto_approve"

	ReviewAfter1Day   = "1_day"
	ReviewAfter1Week  = "1_week"
	ReviewAfter1Month = "1_month"
)

type DecisionValue string

const (
	DecisionNoTrade               DecisionValue = "NO_TRADE"
	DecisionWatch                 DecisionValue = "WATCH"
	DecisionSetupForming          DecisionValue = "SETUP_FORMING"
	DecisionTradeCandidate        DecisionValue = "TRADE_CANDIDATE"
	DecisionPaperApprovalRequired DecisionValue = "PAPER_APPROVAL_REQUIRED"
	DecisionRejectedByRisk        DecisionValue = "REJECTED_BY_RISK"
)

type Decision struct {
	DecisionID             string        `json:"decision_id"`
	EventID                string        `json:"event_id"`
	Brain                  string        `json:"brain"`
	Decision               DecisionValue `json:"decision"`
	ConfidenceScore        float64       `json:"confidence_score"`
	ClarityScore           float64       `json:"clarity_score"`
	EdgeScore              float64       `json:"edge_score"`
	ConflictScore          float64       `json:"conflict_score"`
	RiskScore              float64       `json:"risk_score"`
	PrimaryReason          string        `json:"primary_reason"`
	SupportingReasons      []string      `json:"supporting_reasons"`
	RequiredConfirmations  []string      `json:"required_confirmations"`
	InvalidationConditions []string      `json:"invalidation_conditions"`
	AllowedActions         []string      `json:"allowed_actions"`
	ForbiddenActions       []string      `json:"forbidden_actions"`
	ReviewAfter            []string      `json:"review_after"`
}

func AllowedDecisions() []DecisionValue {
	return []DecisionValue{
		DecisionNoTrade,
		DecisionWatch,
		DecisionSetupForming,
		DecisionTradeCandidate,
		DecisionPaperApprovalRequired,
		DecisionRejectedByRisk,
	}
}

func (d Decision) IsError() bool {
	return false
}

func defaultForbiddenActions() []string {
	return []string{ActionExecuteTrade, ActionCreateLiveOrder, ActionAutoApprove}
}

func defaultReviewWindows() []string {
	return []string{ReviewAfter1Day, ReviewAfter1Week, ReviewAfter1Month}
}
