package profitability

import "time"

type Trend string

const (
	TrendUp      Trend = "up"
	TrendDown    Trend = "down"
	TrendFlat    Trend = "flat"
	TrendUnknown Trend = "unknown"
)

type Direction string

const (
	DirectionUp      Direction = "up"
	DirectionDown    Direction = "down"
	DirectionFlat    Direction = "flat"
	DirectionUnknown Direction = "unknown"
)

type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

type MarketRegime string

const (
	RegimeRiskOn            MarketRegime = "risk_on"
	RegimeRiskOff           MarketRegime = "risk_off"
	RegimeHighVolatility    MarketRegime = "high_volatility"
	RegimeLowVolatility     MarketRegime = "low_volatility"
	RegimeRatesDominant     MarketRegime = "rates_dominant"
	RegimeGrowthDominant    MarketRegime = "growth_dominant"
	RegimeInflationFear     MarketRegime = "inflation_fear"
	RegimeRecessionFear     MarketRegime = "recession_fear"
	RegimeLiquidityStress   MarketRegime = "liquidity_stress"
	RegimeTechMomentum      MarketRegime = "tech_momentum"
	RegimeDefensiveRotation MarketRegime = "defensive_rotation"
	RegimeUnclear           MarketRegime = "unclear"
)

type AssetState struct {
	Trend         Trend
	AboveMA20     bool
	AboveMA50     bool
	RelativeToSPY float64
	MovePercent   float64
}

type MarketRegimeInput struct {
	AsOfUTC time.Time
	Assets  map[string]AssetState
}

type MarketRegimeSnapshot struct {
	AsOfUTC          time.Time
	PrimaryRegime    MarketRegime
	SecondaryRegimes []MarketRegime
	Confidence       float64
	Inputs           map[string]AssetState
	MissingInputs    []string
	Reasons          []string
}

type CrossAssetVerdict string

const (
	CrossAssetConfirmed          CrossAssetVerdict = "confirmed"
	CrossAssetPartiallyConfirmed CrossAssetVerdict = "partially_confirmed"
	CrossAssetConflicted         CrossAssetVerdict = "conflicted"
	CrossAssetInsufficientData   CrossAssetVerdict = "insufficient_data"
	CrossAssetNotConfirmed       CrossAssetVerdict = "not_confirmed"
)

type CrossAssetInput struct {
	MacroEventID string
	PlaybookKey  string
	AsOfUTC      time.Time
	Expected     map[string]Direction
	Observed     map[string]Direction
}

type CrossAssetResult struct {
	MacroEventID      string
	PlaybookKey       string
	AsOfUTC           time.Time
	ConfirmationScore float64
	Verdict           CrossAssetVerdict
	AssetResults      map[string]Direction
	Conflicts         []string
	MissingAssets     []string
	Reasons           []string
}

type CalendarDirection string

const (
	CalendarDirectionHawkishRates CalendarDirection = "hawkish_rates"
	CalendarDirectionDovishRates  CalendarDirection = "dovish_rates"
	CalendarDirectionGrowthStrong CalendarDirection = "growth_strong"
	CalendarDirectionGrowthWeak   CalendarDirection = "growth_weak"
	CalendarDirectionNeutral      CalendarDirection = "neutral"
)

type CalendarEventInput struct {
	Provider        string
	ProviderEventID string
	EventType       string
	Country         string
	ReleaseTimeUTC  time.Time
	Actual          *float64
	Forecast        *float64
	Previous        *float64
	RevisedPrevious *float64
	Unit            string
	Importance      string
	SourceURL       string
	RawPayload      map[string]any
	NowUTC          time.Time
}

type CalendarEvent struct {
	CalendarEventInput
	SurpriseValue   float64
	SurprisePercent float64
	Direction       CalendarDirection
}

type ValidationResult struct {
	Valid  bool
	Status string
	Reason string
}

type ConfounderType string

const (
	ConfounderSameTimeMacro     ConfounderType = "same_time_macro_release"
	ConfounderFedSpeaker        ConfounderType = "fed_speaker"
	ConfounderTreasuryAuction   ConfounderType = "treasury_auction"
	ConfounderMegaCapEarnings   ConfounderType = "mega_cap_earnings"
	ConfounderSectorNews        ConfounderType = "sector_specific_news"
	ConfounderGeopoliticalShock ConfounderType = "geopolitical_shock"
	ConfounderOilShock          ConfounderType = "oil_shock"
	ConfounderCreditEvent       ConfounderType = "credit_event"
	ConfounderBrokerOrDataIssue ConfounderType = "broker/data_issue"
)

type ConfounderImpact string

const (
	ConfounderBlocksTrade       ConfounderImpact = "blocks_trade"
	ConfounderReducesConfidence ConfounderImpact = "reduces_confidence"
	ConfounderRequiresManual    ConfounderImpact = "requires_manual_review"
	ConfounderReassignsCause    ConfounderImpact = "reassigns_cause"
	ConfounderInformationalOnly ConfounderImpact = "informational_only"
)

type ConfounderEvent struct {
	ID              string
	RelatedEventID  string
	Type            ConfounderType
	AffectedSymbols []string
	Headline        string
	EventTimeUTC    time.Time
	Severity        Severity
	Confidence      float64
	Reason          string
	Source          string
	RawPayload      map[string]any
}

type ConfounderInput struct {
	PrimaryEventID string
	PrimaryTimeUTC time.Time
	PrimarySymbols []string
	NearbyEvents   []ConfounderEvent
}

type ConfounderLink struct {
	EventID           string
	ConfounderEventID string
	Impact            ConfounderImpact
	Reason            string
	Confounder        ConfounderEvent
}

type ExecutionVerdict string

const (
	ExecutionGood             ExecutionVerdict = "good"
	ExecutionAcceptable       ExecutionVerdict = "acceptable"
	ExecutionPoor             ExecutionVerdict = "poor"
	ExecutionBlocked          ExecutionVerdict = "blocked"
	ExecutionInsufficientData ExecutionVerdict = "insufficient_data"
)

type ExecutionQualityInput struct {
	Symbol                     string
	AsOfUTC                    time.Time
	EventTimeUTC               time.Time
	SpreadPercent              *float64
	SlippageEstimatePercent    *float64
	VolumeRatio                float64
	MarketDataFresh            bool
	BrokerAvailable            bool
	EventVolatilityState       string
	EventNoTradeDelaySeconds   int
	MaxSpreadPercent           float64
	MaxSlippageEstimatePercent float64
	MinVolumeRatio             float64
	RawPayload                 map[string]any
}

type ExecutionQualitySnapshot struct {
	Symbol                  string
	AsOfUTC                 time.Time
	SpreadPercent           *float64
	VolumeOK                bool
	SlippageEstimatePercent *float64
	MarketDataFresh         bool
	BrokerAvailable         bool
	EventVolatilityState    string
	Verdict                 ExecutionVerdict
	Reasons                 []string
	RawPayload              map[string]any
}

type PositionVerdict string

const (
	PositionAllowed          PositionVerdict = "allowed"
	PositionReduced          PositionVerdict = "reduced"
	PositionBlocked          PositionVerdict = "blocked"
	PositionInsufficientData PositionVerdict = "insufficient_data"
)

type PositionSizingInput struct {
	CandidateID             string
	Symbol                  string
	AccountEquity           float64
	EntryPrice              float64
	StopPrice               float64
	RequestedRiskPct        float64
	MaxRiskPct              float64
	MaxDailyLossPct         float64
	MaxWeeklyLossPct        float64
	CurrentDailyLossPct     float64
	CurrentWeeklyLossPct    float64
	SameThemeExposureCount  int
	CorrelatedExposureCount int
	Confidence              float64
	MarketRegime            MarketRegime
}

type PositionSizeRecommendation struct {
	CandidateID         string
	Symbol              string
	AccountEquity       float64
	EntryPrice          float64
	StopPrice           float64
	RiskPercent         float64
	CashRisk            float64
	PositionSize        float64
	AdjustedRiskPercent float64
	Adjustments         []string
	Verdict             PositionVerdict
	Reasons             []string
}

type StrategyResult string

const (
	StrategyMatchedAllowed StrategyResult = "matched_allowed"
	StrategyMatchedWatch   StrategyResult = "matched_watch_only"
	StrategyMatchedBlocked StrategyResult = "matched_blocked"
	StrategyNoMatch        StrategyResult = "no_strategy_match"
)

type StrategyPlaybookInput struct {
	EventType         string
	Symbol            string
	FundamentalScore  float64
	TechnicalScore    float64
	Regime            MarketRegime
	CrossAssetVerdict CrossAssetVerdict
	ExecutionVerdict  ExecutionVerdict
	PositionVerdict   PositionVerdict
	MinutesAfterEvent int
	BacktestStatus    string
}

type StrategyPlaybookResult struct {
	PlaybookKey string
	Matched     bool
	Result      StrategyResult
	Reasons     []string
	FailedRules []string
}

type WalkAwayCategory string

const (
	WalkAwayCrossAssetConflict WalkAwayCategory = "cross_asset_conflict"
	WalkAwayRegimeConflict     WalkAwayCategory = "regime_conflict"
	WalkAwayPoorLiquidity      WalkAwayCategory = "poor_liquidity"
	WalkAwayRiskLimitHit       WalkAwayCategory = "risk_limit_hit"
	WalkAwayStrategyNotMatched WalkAwayCategory = "strategy_not_matched"
	WalkAwayMissingData        WalkAwayCategory = "missing_data"
)

type WalkAwaySeverity string

const (
	WalkAwayInfo     WalkAwaySeverity = "info"
	WalkAwayWarning  WalkAwaySeverity = "warning"
	WalkAwayBlocker  WalkAwaySeverity = "blocker"
	WalkAwayCritical WalkAwaySeverity = "critical"
)

type WalkAwayInput struct {
	EventID    string
	Symbol     string
	Regime     MarketRegimeSnapshot
	CrossAsset CrossAssetResult
	Execution  ExecutionQualitySnapshot
	Position   PositionSizeRecommendation
	Strategy   StrategyPlaybookResult
}

type WalkAwayDecision struct {
	EventID      string
	Symbol       string
	Category     WalkAwayCategory
	Severity     WalkAwaySeverity
	Reason       string
	EvidenceRefs map[string]any
}

type CandidateGateInput struct {
	Regime      MarketRegimeSnapshot
	CrossAsset  CrossAssetResult
	Confounders []ConfounderLink
	Execution   ExecutionQualitySnapshot
	Position    PositionSizeRecommendation
	Strategy    StrategyPlaybookResult
	WalkAways   []WalkAwayDecision
}

type CandidateGateDecision struct {
	Allowed  bool
	Blockers []string
	Warnings []string
}

type PriceCandle struct {
	High float64
	Low  float64
}

type TradeOutcome string

const (
	TradeOutcomeWin                    TradeOutcome = "win"
	TradeOutcomeLoss                   TradeOutcome = "loss"
	TradeOutcomeBreakeven              TradeOutcome = "breakeven"
	TradeOutcomeAvoidedGoodTrade       TradeOutcome = "avoided_good_trade"
	TradeOutcomeAvoidedBadTrade        TradeOutcome = "avoided_bad_trade"
	TradeOutcomeInvalidatedBeforeEntry TradeOutcome = "invalidated_before_entry"
	TradeOutcomeManualRejectCorrect    TradeOutcome = "manual_reject_correct"
	TradeOutcomeManualRejectIncorrect  TradeOutcome = "manual_reject_incorrect"
)

type TradeReviewInput struct {
	CandidateID string
	Symbol      string
	StrategyKey string
	EntryPrice  float64
	ExitPrice   float64
	StopPrice   float64
	TargetPrice float64
	Candles     []PriceCandle
	Outcome     TradeOutcome
}

type TradeReview struct {
	CandidateID string
	Symbol      string
	StrategyKey string
	EntryPrice  float64
	ExitPrice   float64
	StopPrice   float64
	TargetPrice float64
	MFER        float64
	MAER        float64
	FinalR      float64
	Outcome     TradeOutcome
}

type PerformanceInput struct {
	EventsAnalyzed int
	Candidates     int
	WalkAways      int
	Reviews        []TradeReview
}

type EventFunnelMetrics struct {
	EventsAnalyzed int
	Candidates     int
	WalkAways      int
	CandidateRate  float64
}

type StrategyPerformance struct {
	StrategyKey string
	Trades      int
	WinRate     float64
	AverageR    float64
	Expectancy  float64
}

type PerformanceDashboard struct {
	EventFunnel         EventFunnelMetrics
	StrategyPerformance []StrategyPerformance
}

type SimulationVerdict string

const (
	SimulationAcceptable         SimulationVerdict = "acceptable"
	SimulationReduceRisk         SimulationVerdict = "reduce_risk"
	SimulationDisableStrategy    SimulationVerdict = "disable_strategy"
	SimulationInsufficientSample SimulationVerdict = "insufficient_sample"
)

type RiskSimulationInput struct {
	StrategyKey     string
	RMultiples      []float64
	SimulationCount int
	RiskPerTradePct float64
	MinSampleSize   int
}

type RiskSimulationResult struct {
	StrategyKey               string
	SimulationCount           int
	SampleSize                int
	AverageR                  float64
	MaxDrawdownR              float64
	MaxLossStreak             int
	RiskOfRuin                float64
	ProbabilityEndingNegative float64
	Verdict                   SimulationVerdict
	Warnings                  []string
}
