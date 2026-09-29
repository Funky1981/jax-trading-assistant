package portfoliorisk

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strings"
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

// legacyPortfolioSnapshotV1 is the exact pre-KnownEmpty JSON field contract.
// Keep it local to the test so the expected identity does not use the current
// PortfolioSnapshot serializer.
type legacyPortfolioSnapshotV1 struct {
	SnapshotID        string         `json:"snapshot_id"`
	ContractVersion   string         `json:"contract_version"`
	IdentityAlgorithm string         `json:"identity_algorithm"`
	AccountID         string         `json:"account_id"`
	AsOf              time.Time      `json:"as_of"`
	CapturedAt        time.Time      `json:"captured_at"`
	Provider          string         `json:"provider"`
	Currency          string         `json:"currency"`
	Synthetic         bool           `json:"synthetic"`
	Cash              ObservedNumber `json:"cash"`
	Equity            ObservedNumber `json:"equity"`
	Positions         []Position     `json:"positions"`
	Provenance        []string       `json:"provenance"`
	ValuationBasis    string         `json:"valuation_basis"`
}

func legacySnapshotFromCurrent(snapshot PortfolioSnapshot) legacyPortfolioSnapshotV1 {
	snapshot.SnapshotID = ""
	if snapshot.ContractVersion == "" {
		snapshot.ContractVersion = PortfolioContractVersion
	}
	if snapshot.IdentityAlgorithm == "" {
		snapshot.IdentityAlgorithm = PortfolioIdentityAlgorithm
	}
	snapshot.AsOf = snapshot.AsOf.UTC()
	snapshot.CapturedAt = snapshot.CapturedAt.UTC()
	snapshot.Currency = strings.ToUpper(strings.TrimSpace(snapshot.Currency))
	snapshot.Provider = strings.TrimSpace(snapshot.Provider)
	snapshot.AccountID = strings.TrimSpace(snapshot.AccountID)
	snapshot.ValuationBasis = strings.TrimSpace(snapshot.ValuationBasis)
	snapshot.Provenance = append([]string(nil), snapshot.Provenance...)
	sort.Strings(snapshot.Provenance)
	positions := append([]Position(nil), snapshot.Positions...)
	for i := range positions {
		positions[i].InstrumentID = strings.TrimSpace(positions[i].InstrumentID)
		positions[i].Currency = strings.ToUpper(strings.TrimSpace(positions[i].Currency))
		positions[i].PriceSource = strings.TrimSpace(positions[i].PriceSource)
		positions[i].Provenance = append([]string(nil), positions[i].Provenance...)
		sort.Strings(positions[i].Provenance)
		positions[i].ValuationAsOf = positions[i].ValuationAsOf.UTC()
	}
	sort.Slice(positions, func(i, j int) bool {
		if positions[i].InstrumentID != positions[j].InstrumentID {
			return positions[i].InstrumentID < positions[j].InstrumentID
		}
		return positions[i].SignedQuantity < positions[j].SignedQuantity
	})
	return legacyPortfolioSnapshotV1{
		ContractVersion: snapshot.ContractVersion, IdentityAlgorithm: snapshot.IdentityAlgorithm,
		AccountID: snapshot.AccountID, AsOf: snapshot.AsOf, CapturedAt: snapshot.CapturedAt,
		Provider: snapshot.Provider, Currency: snapshot.Currency, Synthetic: snapshot.Synthetic,
		Cash: snapshot.Cash, Equity: snapshot.Equity, Positions: positions,
		Provenance: snapshot.Provenance, ValuationBasis: snapshot.ValuationBasis,
	}
}

func legacySnapshotIdentity(snapshot legacyPortfolioSnapshotV1) (string, []byte, error) {
	payload, err := json.Marshal(snapshot)
	if err != nil {
		return "", nil, err
	}
	digest := sha256.Sum256(payload)
	return "pf_" + hex.EncodeToString(digest[:]), payload, nil
}

func TestKnownEmptyFalsePreservesLegacyV1SnapshotIdentity(t *testing.T) {
	snapshot := fixtureSnapshot()
	snapshot.Positions = append(snapshot.Positions,
		Position{InstrumentID: "NYSE:XYZ", InstrumentResolved: true, Currency: "USD", SignedQuantity: -2,
			Price: KnownNumber(50, "fixture"), MarketValue: KnownNumber(-100, "fixture"), CostBasis: UnknownNumber("not supplied"),
			ValuationAsOf: stateNow.Add(-3 * time.Minute), PriceSource: "fixture", Provenance: []string{"fixture:z", "fixture:a"}},
	)
	snapshot.Provenance = []string{"fixture:z", "fixture:a"}

	legacy := legacySnapshotFromCurrent(snapshot)
	legacyID, legacyJSON, err := legacySnapshotIdentity(legacy)
	if err != nil {
		t.Fatal(err)
	}
	current, err := BuildSnapshot(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if current.SnapshotID != legacyID {
		t.Fatalf("current v1 identity %q differs from pre-KnownEmpty identity %q; legacy JSON: %s", current.SnapshotID, legacyID, legacyJSON)
	}
	if strings.Contains(string(legacyJSON), `"known_empty"`) {
		t.Fatal("legacy canonical JSON unexpectedly contains known_empty")
	}
}

func TestLegacySnapshotPayloadWithoutKnownEmptyReloadsWithSameIdentity(t *testing.T) {
	legacy := legacySnapshotFromCurrent(fixtureSnapshot())
	legacyID, _, err := legacySnapshotIdentity(legacy)
	if err != nil {
		t.Fatal(err)
	}
	legacy.SnapshotID = legacyID
	payload, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(payload), `"known_empty"`) {
		t.Fatal("historical payload contains known_empty")
	}
	var decoded PortfolioSnapshot
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.KnownEmpty {
		t.Fatal("missing historical known_empty was not decoded as false")
	}
	canonical, err := BuildSnapshot(decoded)
	if err != nil {
		t.Fatalf("rebuild historical snapshot: %v", err)
	}
	if canonical.SnapshotID != legacyID {
		t.Fatalf("historical identity changed: got %q want %q", canonical.SnapshotID, legacyID)
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

func TestKnownEmptyPortfolioRequiresExplicitKnownFactsAndProvenance(t *testing.T) {
	snapshot := PortfolioSnapshot{
		AccountID: "empty-paper-account", AsOf: stateNow.Add(-time.Minute), CapturedAt: stateNow,
		Provider: "paper-ledger+account", Currency: "USD", ValuationBasis: "cash-plus-marked-positions",
		Cash: KnownNumber(100000, "paper_accounts.cash"), Equity: KnownNumber(100000, "derived-cash-plus-market-value"),
		Provenance: []string{"paper_accounts:empty-paper-account", "paper_ledger_events:empty-paper-account"},
	}
	if _, err := BuildSnapshot(snapshot); !errors.Is(err, ErrInvalidSnapshot) {
		t.Fatalf("unmarked zero-position snapshot error=%v, want invalid", err)
	}
	snapshot.KnownEmpty = true
	canonical, err := BuildSnapshot(snapshot)
	if err != nil {
		t.Fatalf("explicit known-empty snapshot: %v", err)
	}
	if len(canonical.Positions) != 0 || !canonical.KnownEmpty || canonical.Synthetic {
		t.Fatalf("known-empty snapshot fabricated or misclassified positions: %#v", canonical)
	}
	knownEmptyJSON, err := json.Marshal(canonical)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(knownEmptyJSON), `"known_empty":true`) {
		t.Fatalf("true known-empty semantic was not explicitly serialized: %s", knownEmptyJSON)
	}
	var knownEmptyRoundTrip PortfolioSnapshot
	if err := json.Unmarshal(knownEmptyJSON, &knownEmptyRoundTrip); err != nil || !knownEmptyRoundTrip.KnownEmpty {
		t.Fatalf("known-empty round trip lost true semantic: snapshot=%+v err=%v", knownEmptyRoundTrip, err)
	}
	if _, err := BuildSnapshot(knownEmptyRoundTrip); err != nil {
		t.Fatalf("known-empty round trip identity invalid: %v", err)
	}
	nonEmptyKnownEmpty := fixtureSnapshot()
	nonEmptyKnownEmpty.KnownEmpty = true
	if _, err := BuildSnapshot(nonEmptyKnownEmpty); !errors.Is(err, ErrInvalidSnapshot) {
		t.Fatalf("non-empty KnownEmpty=true error=%v, want invalid", err)
	}
	for name, mutate := range map[string]func(*PortfolioSnapshot){
		"unknown-cash":        func(s *PortfolioSnapshot) { s.Cash = UnknownNumber("unknown") },
		"unknown-equity":      func(s *PortfolioSnapshot) { s.Equity = UnknownNumber("unknown") },
		"non-positive-equity": func(s *PortfolioSnapshot) { s.Equity = KnownNumber(0, "fixture") },
		"missing-provenance":  func(s *PortfolioSnapshot) { s.Provenance = nil },
	} {
		t.Run(name, func(t *testing.T) {
			invalid := snapshot
			mutate(&invalid)
			if _, err := BuildSnapshot(invalid); !errors.Is(err, ErrInvalidSnapshot) {
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
