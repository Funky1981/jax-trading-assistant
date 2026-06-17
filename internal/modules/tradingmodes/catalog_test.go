package tradingmodes

import "testing"

func TestDefaultCatalogIncludesETFNewsPaperMode(t *testing.T) {
	catalog := DefaultCatalog()
	mode, ok := catalog.Get("etf_news_paper")
	if !ok {
		t.Fatalf("expected etf_news_paper mode in catalog")
	}
	if mode.AssetClass != "ETF" {
		t.Fatalf("asset class = %q, want ETF", mode.AssetClass)
	}
	if mode.ExecutionPolicy != "candidate_approval_only" {
		t.Fatalf("execution policy = %q, want candidate_approval_only", mode.ExecutionPolicy)
	}
	if mode.RuntimeMode != "paper" {
		t.Fatalf("runtime mode = %q, want paper", mode.RuntimeMode)
	}
	if !mode.RiskDefaults.ApprovalRequired {
		t.Fatal("ApprovalRequired must be true")
	}

	want := []string{
		"etf_news_market_panic_reversal_v1",
		"etf_news_sector_momentum_v1",
		"etf_news_rates_bonds_rotation_v1",
	}
	got := make(map[string]bool, len(mode.Strategies))
	for _, s := range mode.Strategies {
		got[s.StrategyTypeID] = true
	}
	for _, id := range want {
		if !got[id] {
			t.Fatalf("missing strategy %s", id)
		}
	}
}

func TestDefaultCatalogIncludesAllETFUniverse(t *testing.T) {
	catalog := DefaultCatalog()
	mode, ok := catalog.Get("etf_news_paper")
	if !ok {
		t.Fatalf("expected etf_news_paper mode")
	}
	universe := map[string]bool{}
	for _, sym := range mode.Universe {
		universe[sym] = true
	}
	required := []string{"SPY", "QQQ", "DIA", "IWM", "XLK", "XLF", "XLE", "SMH", "SOXX", "TLT", "GLD"}
	for _, sym := range required {
		if !universe[sym] {
			t.Fatalf("missing ETF %s in universe", sym)
		}
	}
}

func TestDefaultCatalogIncludesResearchOnlyMode(t *testing.T) {
	catalog := DefaultCatalog()
	mode, ok := catalog.Get("research_only")
	if !ok {
		t.Fatalf("expected research_only mode in catalog")
	}
	if mode.ExecutionPolicy != "no_execution" {
		t.Fatalf("execution policy = %q, want no_execution", mode.ExecutionPolicy)
	}
}

func TestDefaultCatalogIncludesSwingResearchMode(t *testing.T) {
	catalog := DefaultCatalog()
	mode, ok := catalog.Get("etf_swing_research")
	if !ok {
		t.Fatalf("expected etf_swing_research mode")
	}
	if mode.RuntimeMode != "research" {
		t.Fatalf("runtime mode = %q, want research", mode.RuntimeMode)
	}
	if mode.ExecutionPolicy != "no_execution" {
		t.Fatalf("execution policy = %q, want no_execution", mode.ExecutionPolicy)
	}
	if !mode.RiskDefaults.ApprovalRequired {
		t.Fatal("swing research must remain approval-gated before promotion")
	}
}

func TestDefaultCatalogIncludesSwingPaperMode(t *testing.T) {
	catalog := DefaultCatalog()
	mode, ok := catalog.Get("etf_swing_paper")
	if !ok {
		t.Fatalf("expected etf_swing_paper mode")
	}
	if mode.RuntimeMode != "paper" {
		t.Fatalf("runtime mode = %q, want paper", mode.RuntimeMode)
	}
	if mode.ExecutionPolicy != "candidate_approval_only" {
		t.Fatalf("execution policy = %q, want candidate_approval_only", mode.ExecutionPolicy)
	}
	if mode.RiskDefaults.FlattenBy != "daily_revalidation" {
		t.Fatalf("flatten policy = %q, want daily_revalidation", mode.RiskDefaults.FlattenBy)
	}
}

func TestCatalogGetUnknownMode(t *testing.T) {
	catalog := DefaultCatalog()
	_, ok := catalog.Get("does_not_exist")
	if ok {
		t.Fatal("expected false for unknown mode")
	}
}
