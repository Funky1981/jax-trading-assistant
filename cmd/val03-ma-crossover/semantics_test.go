package main

import "testing"

func TestVAL03R1ExecutionSemantics(t *testing.T) {
	if entryIndexForSignal(10) != 11 {
		t.Fatal("entry must be the next session")
	}
	if entryIndexForSignal(10) == 10 {
		t.Fatal("same-session entry is possible")
	}
	if geometryValid(10, 10, 12) || geometryValid(10, 12, 12) {
		t.Fatal("pre-entry boundary must invalidate an episode")
	}
	if got, ok := openingCrossExit(9, 10, 12); !ok || got != 9 {
		t.Fatal("opening stop gap was not resolved at the open")
	}
	if got, ok := openingCrossExit(13, 10, 12); !ok || got != 13 {
		t.Fatal("opening target gap was not resolved at the open")
	}
	if got, ok := intrabarExit(9, 13, 10, 12); !ok || got != 10 {
		t.Fatal("same-bar ordering is not conservative stop-first")
	}
	if got := quantityForCapital(10000, 3333); got != 3 {
		t.Fatalf("whole-share quantity = %d, want 3", got)
	}
	if got := quantityForCapital(10000, 20000); got != 0 {
		t.Fatalf("quantity floor = %d, want sizing abstention", got)
	}
}

func TestVAL03R1IndicatorHasNoFutureLookahead(t *testing.T) {
	d := &instrumentData{Dates: []string{"a", "b", "c"}, Split: map[string]bar{
		"a": {Close: 1}, "b": {Close: 2}, "c": {Close: 3},
	}}
	before := avgWindow(d, 1, 2)
	d.Split["c"] = bar{Close: 999}
	if after := avgWindow(d, 1, 2); after != before {
		t.Fatalf("future bar changed current indicator: before=%v after=%v", before, after)
	}
}

func TestVAL03R1CostContracts(t *testing.T) {
	base := netReturn(100, 101, 100, 4, 5, 0)
	stress := netReturnFixed(100, 101, 100, 5, 10, 15, 0.5)
	if base == stress {
		t.Fatal("stress cost contract did not change arithmetic")
	}
	if netReturn(100, 100, 1, 0, 0, 0) >= 0 {
		t.Fatal("base commission minimum was not applied")
	}
	if quantityForCapital(10000, 0.01) != 1000000 {
		t.Fatal("whole-share rounding changed")
	}
}

func TestVAL03R1PlaceboAndSeedDeterminism(t *testing.T) {
	a := matchedPlaceboScore("manifest", "SPY", "2023-01-03", "2023-02-01")
	if a != matchedPlaceboScore("manifest", "SPY", "2023-01-03", "2023-02-01") {
		t.Fatal("matched placebo selection is not deterministic")
	}
	if a == matchedPlaceboScore("manifest", "SPY", "2023-01-03", "2023-02-02") {
		t.Fatal("matched placebo tie-break does not include candidate date")
	}
	if seedFor("manifest", "|instrument-year-bootstrap-v2|") == seedFor("manifest", "|timestamp-placebo-v1|") {
		t.Fatal("bootstrap and timestamp domains are not separated")
	}
}

func TestVAL03R1BootstrapAndBreadthAreDeterministic(t *testing.T) {
	cfg := frozenTestConfig(t)
	eps := []episode{{ID: "a", Instrument: "SPY", Year: 2023, NetReturn: 0.01}, {ID: "b", Instrument: "QQQ", Year: 2023, NetReturn: -0.01}}
	first := partitionResult{Episodes: append([]episode(nil), eps...)}
	second := partitionResult{Episodes: append([]episode(nil), eps...)}
	applyBootstrap(&first, "manifest", cfg)
	applyBootstrap(&second, "manifest", cfg)
	if first.BootstrapLow != second.BootstrapLow || first.BootstrapHigh != second.BootstrapHigh {
		t.Fatal("bootstrap is not deterministic")
	}
	blocks, instruments, share, regimes := episodeBreadth(eps)
	if blocks != 2 || instruments != 2 || share != 0.5 || regimes != 0 {
		t.Fatalf("unexpected breadth: %d %d %v %d", blocks, instruments, share, regimes)
	}
}

func TestVAL03R1AbstentionAccountingAndOutputMarker(t *testing.T) {
	results := map[string]partitionResult{
		"a": {Abstentions: map[string]int{"HOLD": 2, "no matched placebo": 1}},
		"b": {Abstentions: map[string]int{"HOLD": 1}},
	}
	totals := totalAbstentions(results)
	if totals["HOLD"] != 3 || totals["no matched placebo"] != 1 {
		t.Fatalf("abstention accounting = %+v", totals)
	}
	data := map[string]*instrumentData{}
	for _, symbol := range symbols {
		data[symbol] = &instrumentData{Dates: []string{"2016-01-04"}}
	}
	if qualitySummary(data, true)["performance_output_generated"] != true {
		t.Fatal("future result output can still be marked false")
	}
}
