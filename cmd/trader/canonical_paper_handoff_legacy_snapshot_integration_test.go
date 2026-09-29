package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"jax-trading-assistant/internal/modules/portfoliorisk"
)

func TestCanonicalPaperHandoffPostgresLegacySnapshotReload(t *testing.T) {
	pool := testFrontendAPIPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	now := time.Now().UTC().Truncate(time.Microsecond)
	snapshot, err := portfoliorisk.BuildSnapshot(portfoliorisk.PortfolioSnapshot{
		AccountID: "legacy-handoff-" + uuid.NewString(), AsOf: now.Add(-time.Minute), CapturedAt: now,
		Provider: "legacy-fixture", Currency: "USD", Synthetic: true,
		Cash: portfoliorisk.KnownNumber(60_000, "legacy-fixture"), Equity: portfoliorisk.KnownNumber(100_000, "legacy-fixture"),
		Positions: []portfoliorisk.Position{{InstrumentID: "NYSE:LEGACY", InstrumentResolved: true, Currency: "USD", SignedQuantity: 100,
			Price: portfoliorisk.KnownNumber(400, "legacy-fixture"), MarketValue: portfoliorisk.KnownNumber(40_000, "legacy-fixture"),
			CostBasis: portfoliorisk.UnknownNumber("not supplied"), ValuationAsOf: now.Add(-time.Minute), PriceSource: "legacy-fixture", Provenance: []string{"legacy:position"}}},
		Provenance: []string{"legacy:account"}, ValuationBasis: "cash-plus-marked-positions",
	})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(payload), `"known_empty"`) {
		t.Fatal("legacy fixture payload unexpectedly includes known_empty")
	}
	if _, err := pool.Exec(ctx, `INSERT INTO portfolio_snapshots(snapshot_id,contract_version,identity_algorithm,account_id,as_of,captured_at,provider,currency,synthetic,payload) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, snapshot.SnapshotID, snapshot.ContractVersion, snapshot.IdentityAlgorithm, snapshot.AccountID, snapshot.AsOf, snapshot.CapturedAt, snapshot.Provider, snapshot.Currency, snapshot.Synthetic, payload); err != nil {
		t.Fatal("insert pre-KnownEmpty snapshot payload")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	loaded, err := loadCanonicalSnapshotTx(ctx, tx, snapshot.SnapshotID)
	if err != nil {
		t.Fatalf("canonical handoff rejected legacy snapshot: %v", err)
	}
	if loaded.SnapshotID != snapshot.SnapshotID || loaded.KnownEmpty {
		t.Fatalf("legacy canonical reload changed identity/semantics: got id=%q knownEmpty=%t want id=%q false", loaded.SnapshotID, loaded.KnownEmpty, snapshot.SnapshotID)
	}
}
