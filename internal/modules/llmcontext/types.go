package llmcontext

import "time"

type TaskType string

const (
	TaskEventClassification          TaskType = "event_classification"
	TaskETFMapping                   TaskType = "etf_mapping"
	TaskHistoricalSummary            TaskType = "historical_summary"
	TaskEvidenceBundleSummary        TaskType = "evidence_bundle_summary"
	TaskApprovalSummary              TaskType = "approval_mobile_summary"
	TaskPricedInExplanation          TaskType = "priced_in_explanation"
	TaskComplexConflictingNewsReview TaskType = "complex_conflicting_news_review"
	TaskPostTradeReflection          TaskType = "post_trade_reflection"
	TaskCurrentTradeDecision         TaskType = "current_trade_decision"
	TaskBrokerStatus                 TaskType = "broker_status"
	TaskCompaction                   TaskType = "compaction"
)

type BlockReason string

const (
	BlockReasonNone                  BlockReason = ""
	BlockReasonPerCallBudgetExceeded BlockReason = "per_call_budget_exceeded"
	BlockReasonDailyBudgetExceeded   BlockReason = "daily_budget_exceeded"
	BlockReasonEligibilityFailed     BlockReason = "eligibility_failed"
	BlockReasonSymbolNotAllowlisted  BlockReason = "symbol_not_allowlisted"
	BlockReasonDuplicateEvent        BlockReason = "duplicate_event"
	BlockReasonQuoteStale            BlockReason = "quote_stale"
	BlockReasonSpreadTooWide         BlockReason = "spread_too_wide"
	BlockReasonPricedIn              BlockReason = "priced_in"
	BlockReasonPricedInUnclear       BlockReason = "priced_in_unclear"
	BlockReasonEvidenceMissing       BlockReason = "evidence_missing"
	BlockReasonLiveTradingPath       BlockReason = "live_trading_path"
	BlockReasonBudgetUnavailable     BlockReason = "budget_unavailable"
	BlockReasonRouteDisabled         BlockReason = "route_disabled"
)

type LLMTask struct {
	TaskType          TaskType
	Symbol            string
	EventID           string
	CandidateID       string
	StrategyID        string
	CorrelationID     string
	EventSummary      string
	MarketSnapshot    string
	EvidenceBundle    string
	GuardrailStatus   string
	RetrievedMemories []MemoryArtifact
	ResponseSchema    string
}

type PromptPackage struct {
	TaskType              TaskType
	Provider              string
	Model                 string
	CacheablePrefix       string
	RetrievedMemory       string
	DynamicContext        string
	ResponseSchema        string
	EstimatedInputTokens  int
	EstimatedOutputTokens int
	EstimatedCostUSD      float64
	CorrelationID         string
	EventID               string
	CandidateID           string
	StrategyID            string
	Symbol                string
	CacheEligible         bool
}

type MemoryArtifact struct {
	ID         string
	Summary    string
	SourceIDs  []string
	CreatedAt  time.Time
	Quality    float64
	TaskType   TaskType
	Symbol     string
	StrategyID string
}

type LLMResult struct {
	CorrelationID string
	Text          string
	InputTokens   int
	OutputTokens  int
	CachedTokens  int
	ActualCostUSD float64
}

type ModelRoute struct {
	ModelAlias     string
	Provider       string
	ProviderModel  string
	Enabled        bool
	Paid           bool
	InputUSDPer1K  float64
	OutputUSDPer1K float64
}

type BudgetState struct {
	DailySpendUSD float64
}

type BudgetLimits struct {
	PerCallUSD float64
	DailyUSD   float64
}

type CostDecision struct {
	Allowed     bool
	BlockReason BlockReason
	Reason      string
}

type UsageRecord struct {
	TaskType              TaskType
	ModelAlias            string
	ProviderModel         string
	EstimatedInputTokens  int
	EstimatedOutputTokens int
	ActualInputTokens     int
	ActualOutputTokens    int
	CachedInputTokens     int
	EstimatedCostUSD      float64
	ActualCostUSD         float64
	CacheHit              bool
	VirtualKey            string
	EventID               string
	CandidateID           string
	StrategyID            string
	Symbol                string
	CorrelationID         string
	Blocked               bool
	BlockReason           BlockReason
	CreatedAt             time.Time
}

type CostRollup struct {
	RollupType          string
	RollupKey           string
	EventCount          int
	CandidateCount      int
	ApprovedCount       int
	TotalInputTokens    int
	TotalOutputTokens   int
	TotalCostUSD        float64
	PaidCallsAvoided    int
	HeadroomTokensSaved int
	From                time.Time
	To                  time.Time
}
