package portfoliorisk

import (
	"context"
	"errors"
	"testing"
	"time"
)

var stateNow = time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)

func fixtureSnapshot() PortfolioSnapshot {
	return PortfolioSnapshot{
		AccountID: "fixture-account-09", AsOf: stateNow.Add(-2 * time.Minute), CapturedAt: stateNow.Add(-time.Minute),
		Provider: "jax-phase09-fixture", Currency: "USD", Synthetic: true, ValuationBasis: "frozen-mark-to-market",
		Cash: KnownNumber(60000, "fixture"), Equity: KnownNumber(100000, "fixture"), Provenance: []string{"fixture:account:09"},
		Positions: []Position{
			{InstrumentID: "NYSE:ABC", InstrumentResolved: true, Currency: "USD", SignedQuantity: 100, Price: KnownNumber(200, "fixture"), MarketValue: KnownNumber(20000, "fixture"), CostBasis: UnknownNumber("not supplied"), ValuationAsOf: stateNow.Add(-2 * time.Minute), PriceSource: "fixture", Provenance: []string{"fixture:price:abc"}},
		},
	}
}

func TestBuildSnapshotIsStableAndOrderIndependent(t *testing.T) {
	left := fixtureSnapshot()
	right := fixtureSnapshot()
	right.Positions = append([]Position{{InstrumentID: "NYSE:XYZ", InstrumentResolved: true, Currency: "USD", SignedQuantity: -2, Price: KnownNumber(50, "fixture"), MarketValue: KnownNumber(-100, "fixture"), ValuationAsOf: stateNow.Add(-3 * time.Minute), PriceSource: "fixture"}}, right.Positions...)
	left.Positions = append(left.Positions, right.Positions[0])
	right.Positions = []Position{right.Positions[1], right.Positions[0]}
	a, err := BuildSnapshot(left)
	if err != nil {
		t.Fatal(err)
	}
	b, err := BuildSnapshot(right)
	if err != nil {
		t.Fatal(err)
	}
	if a.SnapshotID != b.SnapshotID {
		t.Fatalf("snapshot identity changed with order: %s != %s", a.SnapshotID, b.SnapshotID)
	}
	if a.Positions[0].Direction() != "LONG" || a.Positions[1].Direction() != "SHORT" {
		t.Fatalf("signed direction lost: %#v", a.Positions)
	}
}

func TestUnknownIsNotZeroAndFreshnessFailsClosed(t *testing.T) {
	snapshot := fixtureSnapshot()
	snapshot.Cash = UnknownNumber("provider omitted")
	canonical, err := BuildSnapshot(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if canonical.Cash.Known || canonical.Cash.Value != 0 {
		t.Fatalf("unknown cash was converted: %#v", canonical.Cash)
	}
	got := canonical.AssessFreshness(stateNow, 10*time.Minute)
	if got.Status != FreshnessUnknown {
		t.Fatalf("freshness = %s, want UNKNOWN", got.Status)
	}
	if got.Reason == "" {
		t.Fatal("unknown freshness needs a reason")
	}
}

func TestFreshnessNegativePaths(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*PortfolioSnapshot)
		want   FreshnessStatus
	}{
		{"stale account", func(s *PortfolioSnapshot) {
			s.AsOf = stateNow.Add(-time.Hour)
			s.CapturedAt = stateNow.Add(-time.Hour + time.Minute)
			s.Positions[0].ValuationAsOf = stateNow.Add(-time.Hour)
		}, FreshnessStale},
		{"stale valuation", func(s *PortfolioSnapshot) { s.Positions[0].ValuationAsOf = stateNow.Add(-time.Hour) }, FreshnessStale},
		{"missing valuation", func(s *PortfolioSnapshot) { s.Positions[0].MarketValue = UnknownNumber("missing") }, FreshnessUnknown},
		{"unresolved instrument", func(s *PortfolioSnapshot) { s.Positions[0].InstrumentResolved = false }, FreshnessUnknown},
		{"unsupported currency", func(s *PortfolioSnapshot) { s.Currency = "XYZ"; s.Positions[0].Currency = "XYZ" }, FreshnessUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := fixtureSnapshot()
			snapshot = mutateAndBuild(t, snapshot, tt.mutate)
			result := snapshot.AssessFreshness(stateNow, 10*time.Minute)
			if result.Status != tt.want {
				t.Fatalf("status=%s want=%s (%s)", result.Status, tt.want, result.Reason)
			}
		})
	}
}

func mutateAndBuild(t *testing.T, snapshot PortfolioSnapshot, mutate func(*PortfolioSnapshot)) PortfolioSnapshot {
	t.Helper()
	mutate(&snapshot)
	got, err := BuildSnapshot(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestSnapshotValidationRejectsConflictingAndMalformedFacts(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*PortfolioSnapshot)
	}{
		{"capture before account as of", func(s *PortfolioSnapshot) { s.CapturedAt = s.AsOf.Add(-time.Second) }},
		{"position valuation after snapshot", func(s *PortfolioSnapshot) { s.Positions[0].ValuationAsOf = s.AsOf.Add(time.Second) }},
		{"nan price", func(s *PortfolioSnapshot) { s.Positions[0].Price = KnownNumber(0, "fixture") }},
		{"bad currency", func(s *PortfolioSnapshot) { s.Currency = "US" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := fixtureSnapshot()
			tt.mutate(&snapshot)
			if _, err := BuildSnapshot(snapshot); err == nil || !errors.Is(err, ErrInvalidSnapshot) {
				t.Fatalf("error=%v, want invalid snapshot", err)
			}
		})
	}
}

func TestMemorySnapshotStoreIsAppendOnly(t *testing.T) {
	store := NewMemorySnapshotStore()
	snapshot, err := BuildSnapshot(fixtureSnapshot())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(context.Background(), snapshot); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Get(context.Background(), snapshot.SnapshotID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.SnapshotID != snapshot.SnapshotID {
		t.Fatal("stored snapshot identity changed")
	}
	changed := snapshot
	changed.Cash = KnownNumber(1, "tamper")
	changed.SnapshotID = snapshot.SnapshotID
	if err := store.Save(context.Background(), changed); err == nil {
		t.Fatal("tampered append was accepted")
	}
}
