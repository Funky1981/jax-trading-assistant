package operations

import "time"

type ReviewOperationsReport struct {
	ReportID                      string    `json:"report_id"`
	GeneratedAt                   time.Time `json:"generated_at"`
	TotalTriageItems              int       `json:"total_triage_items"`
	OpenCount                     int       `json:"open_count"`
	AcceptedCount                 int       `json:"accepted_count"`
	RejectedCount                 int       `json:"rejected_count"`
	DeferredCount                 int       `json:"deferred_count"`
	NeedsMoreEvidenceCount        int       `json:"needs_more_evidence_count"`
	ClosedCount                   int       `json:"closed_count"`
	CriticalCount                 int       `json:"critical_count"`
	HighCount                     int       `json:"high_count"`
	MediumCount                   int       `json:"medium_count"`
	LowCount                      int       `json:"low_count"`
	OverdueCount                  int       `json:"overdue_count"`
	DueCount                      int       `json:"due_count"`
	ResearchGapCount              int       `json:"research_gap_count"`
	MissedOpportunityCount        int       `json:"missed_opportunity_count"`
	RiskVetoTooStrictCount        int       `json:"risk_veto_too_strict_count"`
	PaperSetupFailedCount         int       `json:"paper_setup_failed_count"`
	FollowUpActionCount           int       `json:"follow_up_action_count"`
	ActionsRequiringHumanApproval int       `json:"actions_requiring_human_approval"`
	AutoApplyBlockedCount         int       `json:"auto_apply_blocked_count"`
	ForbiddenActions              []string  `json:"forbidden_actions"`
	Summary                       string    `json:"summary"`
	Warnings                      []string  `json:"warnings"`
}

type ReportOptions struct {
	ReportID    string
	GeneratedAt time.Time
	AsOf        time.Time
}
