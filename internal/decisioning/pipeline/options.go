package pipeline

import (
	"time"

	"jax-trading-assistant/internal/decisioning/brains/swing"
	"jax-trading-assistant/internal/decisioning/core"
	"jax-trading-assistant/internal/decisioning/research"
	"jax-trading-assistant/internal/decisioning/risk"
)

type Options struct {
	AllowMissingResearchEvidence bool `json:"allow_missing_research_evidence"`
}

type SwingInput struct {
	Scores                    core.Scores       `json:"scores"`
	Catalyst                  string            `json:"catalyst"`
	SetupFamily               swing.SetupFamily `json:"setup_family"`
	SupportingReasons         []string          `json:"supporting_reasons"`
	RequiredConfirmations     []string          `json:"required_confirmations"`
	PresentConfirmations      []string          `json:"present_confirmations"`
	MissingConfirmations      []string          `json:"missing_confirmations"`
	InvalidationConditions    []string          `json:"invalidation_conditions"`
	RiskRewardRatio           float64           `json:"risk_reward_ratio"`
	UnresolvedEventRisk       []string          `json:"unresolved_event_risk"`
	UpcomingMajorEventRisk    []string          `json:"upcoming_major_event_risk"`
	MarketSectorAlignmentNote string            `json:"market_sector_alignment_note"`
	Asset                     string            `json:"asset"`
	ProposedEntry             float64           `json:"proposed_entry"`
	ProposedEntryHigh         float64           `json:"proposed_entry_high"`
	ProposedStop              float64           `json:"proposed_stop"`
	ProposedTarget            float64           `json:"proposed_target"`
	MaxPaperPositionSize      float64           `json:"max_paper_position_size"`
	LiquiditySpreadNotes      []string          `json:"liquidity_spread_notes"`
}

type Input struct {
	Event              core.Event                 `json:"event"`
	MarketContext      map[string]string          `json:"market_context,omitempty"`
	PortfolioContext   *risk.PortfolioContext     `json:"portfolio_context,omitempty"`
	CurrentExposure    risk.Exposure              `json:"current_exposure,omitempty"`
	SectorExposure     risk.Exposure              `json:"sector_exposure,omitempty"`
	CorrelatedExposure risk.Exposure              `json:"correlated_exposure,omitempty"`
	ResearchEvidence   *research.BacktestEvidence `json:"research_evidence,omitempty"`
	AccountMode        risk.AccountMode           `json:"account_mode,omitempty"`
	CurrentTime        time.Time                  `json:"current_time,omitempty"`
	Options            Options                    `json:"pipeline_options,omitempty"`
	DecisionScores     core.Scores                `json:"decision_scores"`
	Swing              SwingInput                 `json:"swing"`
}

func (o Options) researchRequired() bool {
	return !o.AllowMissingResearchEvidence
}
