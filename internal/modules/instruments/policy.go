package instruments

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

const (
	DefaultCatalogPath = "config/etf-instruments.json"

	ReasonAllowed              = "allowed"
	ReasonUnknownSymbol        = "unknown_symbol"
	ReasonNotETF               = "not_etf"
	ReasonNotEligible          = "not_eligible"
	ReasonExcludedClass        = "excluded_etf_class"
	ReasonModeNotAllowed       = "mode_not_allowed"
	ReasonQuoteMissing         = "quote_missing"
	ReasonQuoteStale           = "quote_stale"
	ReasonBidAskMissing        = "bid_ask_missing"
	ReasonSpreadTooWide        = "spread_too_wide"
	ReasonOutsideSession       = "outside_regular_trading_hours"
	ReasonStopLossRequired     = "stop_loss_required"
	ReasonFlattenCloseRequired = "flatten_by_close_required"
)

type Catalog struct {
	Version     string       `json:"version"`
	Owner       string       `json:"owner"`
	Policy      Policy       `json:"policy"`
	Instruments []Instrument `json:"instruments"`
	hash        string
	bySymbol    map[string]Instrument
}

type Policy struct {
	Phase                 string   `json:"phase"`
	QuoteFreshnessSeconds int      `json:"quote_freshness_seconds"`
	MaxSpreadBps          float64  `json:"max_spread_bps"`
	MinBidSize            int64    `json:"min_bid_size"`
	MinAskSize            int64    `json:"min_ask_size"`
	SessionTimezone       string   `json:"session_timezone"`
	RegularSessionStart   string   `json:"regular_session_start"`
	RegularSessionEnd     string   `json:"regular_session_end"`
	RequireStopLoss       bool     `json:"require_stop_loss"`
	RequireFlattenByClose bool     `json:"require_flatten_by_close"`
	EntryModes            []string `json:"entry_modes"`
}

type Instrument struct {
	Symbol         string   `json:"symbol"`
	AssetClass     string   `json:"asset_class"`
	InstrumentType string   `json:"instrument_type"`
	TradableModes  []string `json:"tradable_modes"`
	Eligibility    string   `json:"eligibility_state"`
	EffectiveDate  string   `json:"effective_date"`
	ChangeOwner    string   `json:"change_owner"`
	Exclusions     []string `json:"exclusions"`
}

type Evaluation struct {
	Symbol         string         `json:"symbol"`
	Allowed        bool           `json:"allowed"`
	ReasonCode     string         `json:"reasonCode"`
	Reason         string         `json:"reason"`
	CatalogVersion string         `json:"catalogVersion"`
	CatalogHash    string         `json:"catalogHash"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

type QuoteSnapshot struct {
	Symbol    string
	Bid       float64
	Ask       float64
	BidSize   int64
	AskSize   int64
	Timestamp time.Time
}

type SubmissionContext struct {
	Now          time.Time
	QuoteTime    time.Time
	Bid          float64
	Ask          float64
	BidSize      int64
	AskSize      int64
	HasStopLoss  bool
	FlattenByEOD bool
}

func LoadDefaultCatalog() (*Catalog, error) {
	path := strings.TrimSpace(os.Getenv("ETF_INSTRUMENT_CATALOG_PATH"))
	if path == "" {
		path = DefaultCatalogPath
	}
	catalog, err := LoadCatalog(path)
	if err == nil || strings.TrimSpace(os.Getenv("ETF_INSTRUMENT_CATALOG_PATH")) != "" {
		return catalog, err
	}
	for _, fallback := range []string{
		"../" + DefaultCatalogPath,
		"../../" + DefaultCatalogPath,
		"../../../" + DefaultCatalogPath,
	} {
		catalog, err = LoadCatalog(fallback)
		if err == nil {
			return catalog, nil
		}
	}
	return nil, err
}

func LoadCatalog(path string) (*Catalog, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var catalog Catalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return nil, err
	}
	sum := sha256.Sum256(data)
	catalog.hash = hex.EncodeToString(sum[:])
	if err := catalog.validate(); err != nil {
		return nil, err
	}
	return &catalog, nil
}

func (c *Catalog) validate() error {
	if strings.TrimSpace(c.Version) == "" {
		return fmt.Errorf("catalog version required")
	}
	if c.Policy.QuoteFreshnessSeconds <= 0 {
		return fmt.Errorf("quote_freshness_seconds must be positive")
	}
	if c.Policy.MaxSpreadBps <= 0 {
		return fmt.Errorf("max_spread_bps must be positive")
	}
	if c.Policy.SessionTimezone == "" || c.Policy.RegularSessionStart == "" || c.Policy.RegularSessionEnd == "" {
		return fmt.Errorf("regular session policy required")
	}
	c.bySymbol = make(map[string]Instrument, len(c.Instruments))
	for _, inst := range c.Instruments {
		inst.Symbol = strings.ToUpper(strings.TrimSpace(inst.Symbol))
		if inst.Symbol == "" {
			return fmt.Errorf("instrument symbol required")
		}
		if _, exists := c.bySymbol[inst.Symbol]; exists {
			return fmt.Errorf("duplicate instrument %s", inst.Symbol)
		}
		c.bySymbol[inst.Symbol] = inst
	}
	if len(c.bySymbol) == 0 {
		return fmt.Errorf("at least one instrument required")
	}
	return nil
}

func (c *Catalog) Hash() string {
	if c == nil {
		return ""
	}
	return c.hash
}

func (c *Catalog) IsKnownETF(symbol string) bool {
	if c == nil {
		return false
	}
	inst, ok := c.bySymbol[strings.ToUpper(strings.TrimSpace(symbol))]
	if !ok {
		return false
	}
	return strings.EqualFold(inst.InstrumentType, "etf") || strings.EqualFold(inst.InstrumentType, "etn")
}

func (c *Catalog) Evaluate(symbol, mode string) Evaluation {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	base := Evaluation{
		Symbol:         symbol,
		CatalogVersion: c.version(),
		CatalogHash:    c.Hash(),
		Metadata:       map[string]any{"runtimeMode": strings.ToLower(strings.TrimSpace(mode))},
	}
	if c == nil {
		base.ReasonCode = ReasonUnknownSymbol
		base.Reason = "ETF instrument catalog is unavailable."
		return base
	}
	inst, ok := c.bySymbol[symbol]
	if !ok {
		base.ReasonCode = ReasonUnknownSymbol
		base.Reason = "Symbol is not present in the approved ETF instrument catalog."
		return base
	}
	base.Metadata["assetClass"] = inst.AssetClass
	base.Metadata["instrumentType"] = inst.InstrumentType
	base.Metadata["eligibilityState"] = inst.Eligibility
	base.Metadata["exclusions"] = append([]string(nil), inst.Exclusions...)
	base.Metadata["tradableModes"] = append([]string(nil), inst.TradableModes...)

	if !strings.EqualFold(inst.InstrumentType, "etf") {
		base.ReasonCode = ReasonNotETF
		if strings.EqualFold(inst.InstrumentType, "etn") {
			base.ReasonCode = ReasonExcludedClass
		}
		base.Reason = fmt.Sprintf("%s is not an approved plain-vanilla ETF.", symbol)
		return base
	}
	if len(inst.Exclusions) > 0 {
		base.ReasonCode = ReasonExcludedClass
		base.Reason = fmt.Sprintf("%s is excluded by ETF class policy: %s.", symbol, strings.Join(inst.Exclusions, ", "))
		return base
	}
	if !strings.EqualFold(inst.Eligibility, "approved") {
		base.ReasonCode = ReasonNotEligible
		base.Reason = fmt.Sprintf("%s eligibility state is %q, not approved.", symbol, inst.Eligibility)
		return base
	}
	mode = strings.ToLower(strings.TrimSpace(mode))
	if !containsFold(inst.TradableModes, mode) || !containsFold(c.Policy.EntryModes, mode) {
		base.ReasonCode = ReasonModeNotAllowed
		base.Reason = fmt.Sprintf("%s is not tradable in %s mode for ETF phase 1.", symbol, mode)
		return base
	}
	base.Allowed = true
	base.ReasonCode = ReasonAllowed
	base.Reason = fmt.Sprintf("%s is approved for ETF phase-1 %s trading.", symbol, mode)
	return base
}

func (c *Catalog) EvaluateSubmission(symbol, mode string, ctx SubmissionContext) Evaluation {
	eval := c.Evaluate(symbol, mode)
	if !eval.Allowed {
		return eval
	}
	if ctx.Now.IsZero() {
		ctx.Now = time.Now().UTC()
	}
	if ctx.QuoteTime.IsZero() {
		return rejectedFrom(eval, ReasonQuoteMissing, "ETF quote is missing.", nil)
	}
	age := ctx.Now.Sub(ctx.QuoteTime)
	if age < 0 {
		age = 0
	}
	metadata := map[string]any{
		"quoteTimestamp":     ctx.QuoteTime.UTC().Format(time.RFC3339),
		"quoteAgeSeconds":    age.Seconds(),
		"maxQuoteAgeSeconds": c.Policy.QuoteFreshnessSeconds,
		"bid":                ctx.Bid,
		"ask":                ctx.Ask,
		"bidSize":            ctx.BidSize,
		"askSize":            ctx.AskSize,
	}
	if age > time.Duration(c.Policy.QuoteFreshnessSeconds)*time.Second {
		return rejectedFrom(eval, ReasonQuoteStale, "ETF quote is older than the phase-1 freshness threshold.", metadata)
	}
	if ctx.Bid <= 0 || ctx.Ask <= 0 || ctx.BidSize < c.Policy.MinBidSize || ctx.AskSize < c.Policy.MinAskSize {
		return rejectedFrom(eval, ReasonBidAskMissing, "ETF quote must include bid, ask, and non-zero bid/ask size.", metadata)
	}
	mid := (ctx.Bid + ctx.Ask) / 2
	spreadBps := ((ctx.Ask - ctx.Bid) / mid) * 10000
	metadata["spreadBps"] = spreadBps
	metadata["maxSpreadBps"] = c.Policy.MaxSpreadBps
	if spreadBps > c.Policy.MaxSpreadBps {
		return rejectedFrom(eval, ReasonSpreadTooWide, "ETF spread exceeds the phase-1 threshold.", metadata)
	}
	if !c.inRegularSession(ctx.Now) {
		return rejectedFrom(eval, ReasonOutsideSession, "ETF entries are limited to regular trading hours.", metadata)
	}
	if c.Policy.RequireStopLoss && !ctx.HasStopLoss {
		return rejectedFrom(eval, ReasonStopLossRequired, "ETF entries require a stop loss.", metadata)
	}
	if c.Policy.RequireFlattenByClose && !ctx.FlattenByEOD {
		return rejectedFrom(eval, ReasonFlattenCloseRequired, "ETF entries require flatten-by-close controls.", metadata)
	}
	for k, v := range metadata {
		eval.Metadata[k] = v
	}
	return eval
}

func (c *Catalog) ETFList() []Instrument {
	if c == nil {
		return nil
	}
	out := make([]Instrument, 0, len(c.Instruments))
	for _, inst := range c.Instruments {
		if strings.EqualFold(inst.InstrumentType, "etf") {
			out = append(out, inst)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Symbol < out[j].Symbol })
	return out
}

func (c *Catalog) version() string {
	if c == nil {
		return ""
	}
	return c.Version
}

func (c *Catalog) inRegularSession(now time.Time) bool {
	loc, err := time.LoadLocation(c.Policy.SessionTimezone)
	if err != nil {
		loc = time.FixedZone("America/New_York", -5*60*60)
	}
	local := now.In(loc)
	if local.Weekday() == time.Saturday || local.Weekday() == time.Sunday {
		return false
	}
	start, err := parseClock(c.Policy.RegularSessionStart, local)
	if err != nil {
		return false
	}
	end, err := parseClock(c.Policy.RegularSessionEnd, local)
	if err != nil {
		return false
	}
	return !local.Before(start) && local.Before(end)
}

func parseClock(raw string, day time.Time) (time.Time, error) {
	parsed, err := time.Parse("15:04", raw)
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(day.Year(), day.Month(), day.Day(), parsed.Hour(), parsed.Minute(), 0, 0, day.Location()), nil
}

func rejectedFrom(eval Evaluation, code, reason string, metadata map[string]any) Evaluation {
	eval.Allowed = false
	eval.ReasonCode = code
	eval.Reason = reason
	if eval.Metadata == nil {
		eval.Metadata = map[string]any{}
	}
	for k, v := range metadata {
		eval.Metadata[k] = v
	}
	return eval
}

func containsFold(values []string, want string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), want) {
			return true
		}
	}
	return false
}
