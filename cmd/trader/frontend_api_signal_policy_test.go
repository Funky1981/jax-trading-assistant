package main

import "testing"

func TestLegacySignalApprovalPolicyBlocksCatalogETFs(t *testing.T) {
	blocked, reason, err := legacySignalApprovalETFBlock("SPY")
	if err != nil {
		t.Fatalf("legacy approval policy returned error: %v", err)
	}
	if !blocked {
		t.Fatal("expected SPY legacy signal approval to be blocked")
	}
	if reason != "ETF signal approvals must use the candidate approval workflow" {
		t.Fatalf("reason = %q, want candidate approval workflow message", reason)
	}
}

func TestLegacySignalApprovalPolicyAllowsNonCatalogSymbols(t *testing.T) {
	blocked, reason, err := legacySignalApprovalETFBlock("AAPL")
	if err != nil {
		t.Fatalf("legacy approval policy returned error: %v", err)
	}
	if blocked {
		t.Fatalf("expected AAPL legacy signal approval to remain available, reason = %q", reason)
	}
}
