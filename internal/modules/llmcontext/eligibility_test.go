package llmcontext

import "testing"

func TestEligibilityGateBlocksHardFailures(t *testing.T) {
	gate := EligibilityGate{}

	tests := []struct {
		name string
		in   EligibilityInput
		want BlockReason
	}{
		{
			name: "non allowlisted symbol",
			in:   validEligibilityInput(func(in *EligibilityInput) { in.SymbolAllowlisted = false }),
			want: BlockReasonSymbolNotAllowlisted,
		},
		{
			name: "duplicate event",
			in:   validEligibilityInput(func(in *EligibilityInput) { in.EventDuplicate = true }),
			want: BlockReasonDuplicateEvent,
		},
		{
			name: "stale quote",
			in:   validEligibilityInput(func(in *EligibilityInput) { in.QuoteFresh = false }),
			want: BlockReasonQuoteStale,
		},
		{
			name: "wide spread",
			in:   validEligibilityInput(func(in *EligibilityInput) { in.SpreadAcceptable = false }),
			want: BlockReasonSpreadTooWide,
		},
		{
			name: "priced in",
			in:   validEligibilityInput(func(in *EligibilityInput) { in.PricedInVerdict = PricedInVerdictPricedIn }),
			want: BlockReasonPricedIn,
		},
		{
			name: "unclear priced in verdict",
			in:   validEligibilityInput(func(in *EligibilityInput) { in.PricedInVerdict = PricedInVerdictUnclear }),
			want: BlockReasonPricedInUnclear,
		},
		{
			name: "missing evidence bundle",
			in:   validEligibilityInput(func(in *EligibilityInput) { in.EvidenceBundlePresent = false }),
			want: BlockReasonEvidenceMissing,
		},
		{
			name: "live trading path",
			in:   validEligibilityInput(func(in *EligibilityInput) { in.PaperMode = false }),
			want: BlockReasonLiveTradingPath,
		},
		{
			name: "budget unavailable",
			in:   validEligibilityInput(func(in *EligibilityInput) { in.BudgetAvailable = false }),
			want: BlockReasonBudgetUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := gate.Evaluate(tt.in)
			if got.Eligible {
				t.Fatalf("expected ineligible, got %#v", got)
			}
			if got.BlockedReason != tt.want {
				t.Fatalf("expected %s, got %s", tt.want, got.BlockedReason)
			}
		})
	}
}

func TestEligibilityGateAllowsCompletePaperCandidate(t *testing.T) {
	got := (EligibilityGate{}).Evaluate(validEligibilityInput(nil))
	if !got.Eligible {
		t.Fatalf("expected eligible, got %#v", got)
	}
	if got.AllowedModelRoute != "local-small" {
		t.Fatalf("expected local route, got %q", got.AllowedModelRoute)
	}
}

func validEligibilityInput(mutator func(*EligibilityInput)) EligibilityInput {
	in := EligibilityInput{
		TaskType:              TaskApprovalSummary,
		EventID:               "evt-1",
		Symbol:                "SPY",
		StrategyID:            "strat-1",
		CandidateID:           "cand-1",
		EvidenceBundleID:      "bundle-1",
		EventExists:           true,
		EventDuplicate:        false,
		EventRecent:           true,
		EventTradeable:        true,
		SourceQualityOK:       true,
		SymbolAllowlisted:     true,
		AssetTypeETF:          true,
		PlainVanillaETF:       true,
		PaperMode:             true,
		QuoteFresh:            true,
		SpreadAcceptable:      true,
		MarketSessionOK:       true,
		HaltState:             false,
		ETFMappingExists:      true,
		PricedInVerdict:       PricedInVerdictNotPricedIn,
		ConfounderAnalysisOK:  true,
		EvidenceBundlePresent: true,
		BudgetAvailable:       true,
		ModelRouteEnabled:     true,
		RequestedModelRoute:   "local-small",
	}
	if mutator != nil {
		mutator(&in)
	}
	return in
}
