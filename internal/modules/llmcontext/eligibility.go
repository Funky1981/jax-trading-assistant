package llmcontext

type PricedInVerdict string

const (
	PricedInVerdictNotPricedIn PricedInVerdict = "not_priced_in"
	PricedInVerdictPricedIn    PricedInVerdict = "priced_in"
	PricedInVerdictUnclear     PricedInVerdict = "unclear"
)

type EligibilityInput struct {
	TaskType              TaskType
	EventID               string
	Symbol                string
	StrategyID            string
	CandidateID           string
	EvidenceBundleID      string
	EventExists           bool
	EventDuplicate        bool
	EventRecent           bool
	EventTradeable        bool
	SourceQualityOK       bool
	SymbolAllowlisted     bool
	AssetTypeETF          bool
	PlainVanillaETF       bool
	PaperMode             bool
	QuoteFresh            bool
	SpreadAcceptable      bool
	MarketSessionOK       bool
	HaltState             bool
	ETFMappingExists      bool
	PricedInVerdict       PricedInVerdict
	ConfounderAnalysisOK  bool
	EvidenceBundlePresent bool
	BudgetAvailable       bool
	ModelRouteEnabled     bool
	RequestedModelRoute   string
}

type EligibilityDecision struct {
	Eligible          bool
	AllowedModelRoute string
	Reason            string
	BlockedReason     BlockReason
}

type EligibilityGate struct{}

func (EligibilityGate) Evaluate(in EligibilityInput) EligibilityDecision {
	switch {
	case !in.EventExists, !in.EventRecent, !in.EventTradeable, !in.SourceQualityOK:
		return blocked(BlockReasonEligibilityFailed)
	case !in.SymbolAllowlisted, !in.AssetTypeETF, !in.PlainVanillaETF:
		return blocked(BlockReasonSymbolNotAllowlisted)
	case in.EventDuplicate:
		return blocked(BlockReasonDuplicateEvent)
	case !in.PaperMode:
		return blocked(BlockReasonLiveTradingPath)
	case !in.QuoteFresh:
		return blocked(BlockReasonQuoteStale)
	case !in.SpreadAcceptable:
		return blocked(BlockReasonSpreadTooWide)
	case !in.MarketSessionOK, in.HaltState:
		return blocked(BlockReasonEligibilityFailed)
	case !in.ETFMappingExists, !in.ConfounderAnalysisOK:
		return blocked(BlockReasonEligibilityFailed)
	case in.PricedInVerdict == PricedInVerdictPricedIn:
		return blocked(BlockReasonPricedIn)
	case in.PricedInVerdict == PricedInVerdictUnclear:
		return blocked(BlockReasonPricedInUnclear)
	case !in.EvidenceBundlePresent || in.EvidenceBundleID == "":
		return blocked(BlockReasonEvidenceMissing)
	case !in.BudgetAvailable:
		return blocked(BlockReasonBudgetUnavailable)
	case !in.ModelRouteEnabled:
		return blocked(BlockReasonRouteDisabled)
	}
	route := in.RequestedModelRoute
	if route == "" {
		route = "local-small"
	}
	return EligibilityDecision{Eligible: true, AllowedModelRoute: route, Reason: "Evidence bundle complete and all guardrails passed."}
}

func blocked(reason BlockReason) EligibilityDecision {
	return EligibilityDecision{Eligible: false, BlockedReason: reason}
}
