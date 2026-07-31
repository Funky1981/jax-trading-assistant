package evidencequality

import "time"

const (
	DecisionNoTrade   = "NO_TRADE"
	DecisionWatch     = "WATCH"
	DecisionCandidate = "CANDIDATE"
)

type ProxyRule struct {
	Symbol     string `json:"symbol"`
	Confidence string `json:"confidence"`
	Reason     string `json:"reason"`
}

type BenchmarkRule struct {
	Symbol string `json:"symbol"`
	Reason string `json:"reason"`
}

type Ruleset struct {
	Version                           string                   `json:"version"`
	PrimaryAnchor                     string                   `json:"primary_anchor"`
	MinimumComparisonGroupSize        int                      `json:"minimum_comparison_group_size"`
	BootstrapIterations               int                      `json:"bootstrap_iterations"`
	PermutationIterations             int                      `json:"permutation_iterations"`
	MaximumIntradayAnchorDelayMinutes int                      `json:"maximum_intraday_anchor_delay_minutes"`
	ControlledSourcePrefixes          []string                 `json:"controlled_source_prefixes"`
	ControlledEventIDMarkers          []string                 `json:"controlled_event_id_markers"`
	ControlledHeadlineMarkers         []string                 `json:"controlled_headline_markers"`
	TestHosts                         []string                 `json:"test_hosts"`
	CategoryProxies                   map[string]ProxyRule     `json:"category_proxies"`
	Benchmarks                        map[string]BenchmarkRule `json:"benchmarks"`
}

type Event struct {
	DecisionID             string
	InboxID                string
	NormalizedEventID      string
	SourceEventIdentity    string
	Decision               string
	RulesetVersion         string
	DecisionAt             time.Time
	PublicationAt          time.Time
	CollectionAt           *time.Time
	ReceiptAt              time.Time
	Source                 string
	SourceURL              string
	EventType              string
	Severity               string
	Confidence             float64
	AffectedAssets         []string
	UnknownAssets          bool
	Reasons                []string
	MissingEvidence        []string
	Headline               string
	Summary                string
	PrimarySymbol          string
	SourceName             string
	FeedURL                string
	SourceNativeID         string
	ContentHash            string
	DataSourceType         string
	SourceProvider         string
	IsSynthetic            bool
	SyntheticReason        string
	SubjectID              string
	SubjectType            string
	SubjectCurrentDecision string
	SubjectEventCount      int
	SourceGroupCount       int
	IndependentSourceCount int
	PrimarySourceCount     int
	RepeatedSourceCount    int
}

type Candle struct {
	Symbol                   string    `json:"symbol"`
	Timestamp                time.Time `json:"timestamp"`
	Open                     float64   `json:"open"`
	High                     float64   `json:"high"`
	Low                      float64   `json:"low"`
	Close                    float64   `json:"close"`
	Timeframe                string    `json:"timeframe"`
	Source                   string    `json:"source"`
	TimestampSemantics       string    `json:"timestampSemantics"`
	RegularTradingHours      *bool     `json:"regularTradingHours,omitempty"`
	MarketDataClassification string    `json:"marketDataClassification"`
}

type SafetyCounts struct {
	Approvals             int64 `json:"approvals"`
	CandidateApprovals    int64 `json:"candidateApprovals"`
	PaperTickets          int64 `json:"paperTickets"`
	ExecutionInstructions int64 `json:"executionInstructions"`
	OrderIntents          int64 `json:"orderIntents"`
	BrokerOrders          int64 `json:"brokerOrders"`
	Trades                int64 `json:"trades"`
	Fills                 int64 `json:"fills"`
}

type RuntimeSafety struct {
	RuntimeMode      string  `json:"runtimeMode"`
	AllowLiveTrading bool    `json:"allowLiveTrading"`
	ExecutionEnabled bool    `json:"executionEnabled"`
	ExecutionWorker  bool    `json:"executionWorkerEnabled"`
	BrokerExecution  bool    `json:"brokerExecutionAllowed"`
	MaximumLeverage  float64 `json:"maximumLeverage"`
}

type ExistingOutcomeSummary struct {
	RecordCount int64    `json:"recordCount"`
	Sources     []string `json:"sources"`
	Horizons    []string `json:"horizons"`
}

type Snapshot struct {
	Events           []Event
	Candles          []Candle
	SafetyBefore     SafetyCounts
	SafetyAfter      SafetyCounts
	ExistingOutcomes ExistingOutcomeSummary
}

type Exclusion struct {
	SourceEventIdentity string `json:"sourceEventIdentity"`
	Decision            string `json:"decision"`
	Reason              string `json:"reason"`
}

type Mapping struct {
	Mapped          bool   `json:"mapped"`
	MappingType     string `json:"mappingType"`
	Symbol          string `json:"symbol,omitempty"`
	Confidence      string `json:"confidence,omitempty"`
	Reason          string `json:"reason"`
	Direct          bool   `json:"direct"`
	RulesetVersion  string `json:"rulesetVersion"`
	Benchmark       string `json:"benchmark,omitempty"`
	BenchmarkReason string `json:"benchmarkReason,omitempty"`
}

type EvaluatedEvent struct {
	Event               Event      `json:"-"`
	SourceEventIdentity string     `json:"sourceEventIdentity"`
	Decision            string     `json:"decision"`
	EventType           string     `json:"eventType"`
	SourceName          string     `json:"sourceName"`
	Headline            string     `json:"headline"`
	PublicationAt       time.Time  `json:"publicationAt"`
	CollectionAt        *time.Time `json:"collectionAt,omitempty"`
	ReceiptAt           time.Time  `json:"receiptAt"`
	DecisionAt          time.Time  `json:"decisionAt"`
	Mapping             Mapping    `json:"mapping"`
	SubjectType         string     `json:"subjectType"`
	SubjectEventCount   int        `json:"subjectEventCount"`
	SourceGroupCount    int        `json:"sourceGroupCount"`
	IndependentSources  int        `json:"independentSourceCount"`
	PrimarySources      int        `json:"primarySourceCount"`
	RepeatedSources     int        `json:"repeatedSourceCount"`
	MissingEvidence     []string   `json:"missingEvidence"`
	Outcomes            []Outcome  `json:"outcomes"`
}

type Outcome struct {
	Anchor                     string    `json:"anchor"`
	AnchorAt                   time.Time `json:"anchorAt"`
	EffectiveAnchorAt          time.Time `json:"effectiveAnchorAt"`
	AnchorDelaySeconds         float64   `json:"anchorDelaySeconds"`
	Horizon                    string    `json:"horizon"`
	Symbol                     string    `json:"symbol"`
	Benchmark                  string    `json:"benchmark,omitempty"`
	StartPrice                 float64   `json:"startPrice"`
	EndPrice                   float64   `json:"endPrice"`
	RawReturn                  float64   `json:"rawReturn"`
	AbsoluteRawReturn          float64   `json:"absoluteRawReturn"`
	AbnormalReturn             *float64  `json:"abnormalReturn,omitempty"`
	AbsoluteAbnormalReturn     *float64  `json:"absoluteAbnormalReturn,omitempty"`
	RealisedRange              float64   `json:"realisedRange"`
	MaximumFavourableExcursion float64   `json:"maximumFavourableExcursion"`
	MaximumAdverseExcursion    float64   `json:"maximumAdverseExcursion"`
	CandleCount                int       `json:"candleCount"`
	MarketDataSource           string    `json:"marketDataSource"`
	TimestampSemantics         string    `json:"timestampSemantics"`
}

type CoverageRow struct {
	Symbol                   string    `json:"symbol"`
	Timeframe                string    `json:"timeframe"`
	Source                   string    `json:"source"`
	TimestampSemantics       string    `json:"timestampSemantics"`
	RegularTradingHours      *bool     `json:"regularTradingHours,omitempty"`
	MarketDataClassification string    `json:"marketDataClassification"`
	Count                    int       `json:"count"`
	First                    time.Time `json:"first"`
	Last                     time.Time `json:"last"`
	GapCount                 int       `json:"gapCount"`
}

type MetricRow struct {
	Decision                       string   `json:"decision"`
	Anchor                         string   `json:"anchor"`
	Horizon                        string   `json:"horizon"`
	Count                          int      `json:"count"`
	MedianAbsoluteReturn           *float64 `json:"medianAbsoluteReturn,omitempty"`
	MeanAbsoluteReturn             *float64 `json:"meanAbsoluteReturn,omitempty"`
	MedianAbsoluteAbnormalReturn   *float64 `json:"medianAbsoluteAbnormalReturn,omitempty"`
	MeanAbsoluteAbnormalReturn     *float64 `json:"meanAbsoluteAbnormalReturn,omitempty"`
	ExceedPointFivePercent         *float64 `json:"exceedPointFivePercent,omitempty"`
	ExceedOnePercent               *float64 `json:"exceedOnePercent,omitempty"`
	ExceedTwoPercent               *float64 `json:"exceedTwoPercent,omitempty"`
	MeanMaximumFavourableExcursion *float64 `json:"meanMaximumFavourableExcursion,omitempty"`
	MeanMaximumAdverseExcursion    *float64 `json:"meanMaximumAdverseExcursion,omitempty"`
	MeanRealisedRange              *float64 `json:"meanRealisedRange,omitempty"`
	MedianBootstrap95Low           *float64 `json:"medianBootstrap95Low,omitempty"`
	MedianBootstrap95High          *float64 `json:"medianBootstrap95High,omitempty"`
}

type Comparison struct {
	Anchor       string   `json:"anchor"`
	Horizon      string   `json:"horizon"`
	WatchCount   int      `json:"watchCount"`
	NoTradeCount int      `json:"noTradeCount"`
	MannWhitneyU *float64 `json:"mannWhitneyU,omitempty"`
	PermutationP *float64 `json:"permutationP,omitempty"`
	CliffsDelta  *float64 `json:"cliffsDelta,omitempty"`
	Available    bool     `json:"available"`
	Limitation   string   `json:"limitation,omitempty"`
}

type LatencySummary struct {
	Decision                    string   `json:"decision"`
	Count                       int      `json:"count"`
	PublicationCollectionMedian *float64 `json:"publicationCollectionMedianSeconds,omitempty"`
	CollectionReceiptMedian     *float64 `json:"collectionReceiptMedianSeconds,omitempty"`
	ReceiptDecisionMedian       *float64 `json:"receiptDecisionMedianSeconds,omitempty"`
	MoveBeforeReceiptMedian     *float64 `json:"moveBeforeReceiptMedian,omitempty"`
	MoveAfterReceiptMedian      *float64 `json:"moveAfterReceiptMedian,omitempty"`
}

type BreakdownRow struct {
	Dimension            string   `json:"dimension"`
	Value                string   `json:"value"`
	Decision             string   `json:"decision"`
	Count                int      `json:"count"`
	Mapped               int      `json:"mapped"`
	OutcomeCount         int      `json:"outcomeCount"`
	MedianAbsoluteReturn *float64 `json:"medianAbsoluteReturn,omitempty"`
}

type MissRow struct {
	SourceEventIdentity string   `json:"sourceEventIdentity"`
	Decision            string   `json:"decision"`
	Headline            string   `json:"headline"`
	Reason              string   `json:"reason"`
	MissingEvidence     []string `json:"missingEvidence"`
	Symbol              string   `json:"symbol"`
	Horizon             string   `json:"horizon"`
	AbsoluteMove        float64  `json:"absoluteMove"`
	ProbableCause       string   `json:"probableCause"`
}

type PopulationSummary struct {
	Considered       int            `json:"considered"`
	Included         int            `json:"included"`
	Excluded         int            `json:"excluded"`
	Mapped           int            `json:"mapped"`
	Unmapped         int            `json:"unmapped"`
	Watch            int            `json:"watch"`
	NoTrade          int            `json:"noTrade"`
	Candidate        int            `json:"candidate"`
	FirstPublication *time.Time     `json:"firstPublication,omitempty"`
	LastPublication  *time.Time     `json:"lastPublication,omitempty"`
	FirstReceipt     *time.Time     `json:"firstReceipt,omitempty"`
	LastReceipt      *time.Time     `json:"lastReceipt,omitempty"`
	FirstDecision    *time.Time     `json:"firstDecision,omitempty"`
	LastDecision     *time.Time     `json:"lastDecision,omitempty"`
	DecisionCounts   map[string]int `json:"decisionCounts"`
	CategoryCounts   map[string]int `json:"categoryCounts"`
	SourceCounts     map[string]int `json:"sourceCounts"`
	ExclusionCounts  map[string]int `json:"exclusionCounts"`
}

type HorizonCoverage struct {
	Horizon      string `json:"horizon"`
	OutcomeCount int    `json:"outcomeCount"`
	EventCount   int    `json:"eventCount"`
	Sufficient   bool   `json:"sufficient"`
	Reason       string `json:"reason"`
}

type Report struct {
	RulesetVersion        string                 `json:"rulesetVersion"`
	InputFingerprint      string                 `json:"inputFingerprint"`
	PrimaryAnchor         string                 `json:"primaryAnchor"`
	Population            PopulationSummary      `json:"population"`
	Exclusions            []Exclusion            `json:"exclusions"`
	MarketCoverage        []CoverageRow          `json:"marketCoverage"`
	HorizonCoverage       []HorizonCoverage      `json:"horizonCoverage"`
	Metrics               []MetricRow            `json:"metrics"`
	Comparisons           []Comparison           `json:"comparisons"`
	Latency               []LatencySummary       `json:"latency"`
	Breakdowns            []BreakdownRow         `json:"breakdowns"`
	Misses                []MissRow              `json:"misses"`
	WeakWatches           []MissRow              `json:"weakWatches"`
	EvidenceAccumulation  []BreakdownRow         `json:"evidenceAccumulation"`
	ExistingOutcomes      ExistingOutcomeSummary `json:"existingOutcomes"`
	SafetyBefore          SafetyCounts           `json:"safetyBefore"`
	SafetyAfter           SafetyCounts           `json:"safetyAfter"`
	RuntimeSafety         RuntimeSafety          `json:"runtimeSafety"`
	Conclusion            string                 `json:"conclusion"`
	ProductRecommendation string                 `json:"productRecommendation"`
	Verdict               string                 `json:"verdict"`
	Limitations           []string               `json:"limitations"`
	Events                []EvaluatedEvent       `json:"events"`
}
