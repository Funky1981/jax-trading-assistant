package triage

import "time"

type SourceType string

const (
	SourceMissedOpportunity      SourceType = "MISSED_OPPORTUNITY"
	SourceRiskVetoTooStrict      SourceType = "RISK_VETO_TOO_STRICT"
	SourceRiskVetoHelped         SourceType = "RISK_VETO_HELPED"
	SourcePaperSetupWorked       SourceType = "PAPER_SETUP_WORKED"
	SourcePaperSetupFailed       SourceType = "PAPER_SETUP_FAILED"
	SourceResearchGap            SourceType = "RESEARCH_GAP"
	SourceScoringReview          SourceType = "SCORING_REVIEW"
	SourceConfirmationRuleReview SourceType = "CONFIRMATION_RULE_REVIEW"
	SourceNoTradeRuleReview      SourceType = "NO_TRADE_RULE_REVIEW"
	SourceDataQualityReview      SourceType = "DATA_QUALITY_REVIEW"
	SourceWatchlistReview        SourceType = "WATCHLIST_REVIEW"
)

type Status string

const (
	StatusOpen              Status = "OPEN"
	StatusAccepted          Status = "ACCEPTED"
	StatusRejected          Status = "REJECTED"
	StatusDeferred          Status = "DEFERRED"
	StatusNeedsMoreEvidence Status = "NEEDS_MORE_EVIDENCE"
	StatusClosed            Status = "CLOSED"
)

type Item struct {
	TriageItemID           string     `json:"triage_item_id"`
	SourceType             SourceType `json:"source_type"`
	SourceID               string     `json:"source_id"`
	SourceDecisionID       string     `json:"source_decision_id"`
	EventID                string     `json:"event_id"`
	Asset                  string     `json:"asset"`
	SetupFamily            string     `json:"setup_family"`
	EventType              string     `json:"event_type"`
	Priority               Priority   `json:"priority"`
	Status                 Status     `json:"status"`
	Reason                 string     `json:"reason"`
	EvidenceRefs           []string   `json:"evidence_refs"`
	SuggestedAction        string     `json:"suggested_action"`
	AllowedFollowUpActions []string   `json:"allowed_follow_up_actions"`
	ForbiddenActions       []string   `json:"forbidden_actions"`
	RequiresHumanApproval  bool       `json:"requires_human_approval"`
	AutoApplyAllowed       bool       `json:"auto_apply_allowed"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
	DueAt                  time.Time  `json:"due_at"`
}
