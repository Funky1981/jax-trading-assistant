package replay

import "time"

type ReplayResult struct {
	ReplayID               string    `json:"replay_id"`
	RecordsProcessed       int       `json:"records_processed"`
	NoTradeCount           int       `json:"no_trade_count"`
	WatchCount             int       `json:"watch_count"`
	SetupFormingCount      int       `json:"setup_forming_count"`
	TradeCandidateCount    int       `json:"trade_candidate_count"`
	RejectedByRiskCount    int       `json:"rejected_by_risk_count"`
	PaperApprovedCount     int       `json:"paper_approved_count"`
	CorrectNoTradeCount    int       `json:"correct_no_trade_count"`
	MissedOpportunityCount int       `json:"missed_opportunity_count"`
	AvoidedLossCount       int       `json:"avoided_loss_count"`
	RiskVetoHelpedCount    int       `json:"risk_veto_helped_count"`
	RiskVetoTooStrictCount int       `json:"risk_veto_too_strict_count"`
	PaperSetupWorkedCount  int       `json:"paper_setup_worked_count"`
	PaperSetupFailedCount  int       `json:"paper_setup_failed_count"`
	LessonsGenerated       int       `json:"lessons_generated"`
	Warnings               []string  `json:"warnings"`
	Errors                 []string  `json:"errors"`
	CreatedAt              time.Time `json:"created_at"`
}
