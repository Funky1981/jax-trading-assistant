package artifacts

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	domainArtifacts "jax-trading-assistant/internal/domain/artifacts"
	"jax-trading-assistant/internal/modules/backtest"
	"jax-trading-assistant/libs/strategies"

	"github.com/google/uuid"
)

// ── fake store ────────────────────────────────────────────────────────────────

type fakeArtifactStore struct {
	artifacts []*domainArtifacts.Artifact
	approvals []*domainArtifacts.Approval
	failOn    string // "artifact" | "approval" | ""
}

func (f *fakeArtifactStore) CreateArtifact(ctx context.Context, a *domainArtifacts.Artifact) error {
	if f.failOn == "artifact" {
		return fmt.Errorf("simulated artifact store failure")
	}
	f.artifacts = append(f.artifacts, a)
	return nil
}

func (f *fakeArtifactStore) CreateApproval(ctx context.Context, ap *domainArtifacts.Approval) error {
	if f.failOn == "approval" {
		return fmt.Errorf("simulated approval store failure")
	}
	f.approvals = append(f.approvals, ap)
	return nil
}

// ── builder factories ─────────────────────────────────────────────────────────

func fakeBuilder() (*Builder, *fakeArtifactStore) {
	fs := &fakeArtifactStore{}
	return NewBuilderWithStore(fs), fs
}

// ── test backtest.Result constructors ─────────────────────────────────────────

func makeResult(strategyID string, seed int64, symbols []string) *backtest.Result {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	return &backtest.Result{
		BacktestResult: strategies.BacktestResult{
			StrategyID:     strategyID,
			Symbol:         "",
			StartDate:      base,
			EndDate:        base.Add(30 * 24 * time.Hour),
			InitialCapital: 100_000,
			FinalCapital:   110_000,
			TotalTrades:    20,
			WinningTrades:  13,
			LosingTrades:   7,
			WinRate:        0.65,
			TotalReturnPct: 10.0,
			MaxDrawdown:    0.05,
			SharpeRatio:    1.42,
			ProfitFactor:   1.8,
			AvgR:           0.9,
		},
		Symbols:    symbols,
		Seed:       seed,
		RunID:      fmt.Sprintf("bt_%s_%d", strategyID, seed),
		RunAt:      time.Now(),
		DurationMs: 123,
	}
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestBuilder_HashNotEmpty(t *testing.T) {
	b, _ := fakeBuilder()
	res := makeResult("rsi_momentum", 42, []string{"AAPL"})

	art, err := b.BuildFromBacktest(context.Background(), "1.0.0",
		map[string]any{"period": 14}, res,
		domainArtifacts.RiskProfile{MaxPositionPct: 0.2, MaxDailyLoss: 1000, AllowedOrderTypes: []string{"LMT"}},
		"test-service",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if art.Hash == "" {
		t.Error("artifact hash must not be empty")
	}
	// Hash is bare hex (no prefix) — just verify it looks like a 64-char hex digest.
	if len(art.Hash) != 64 {
		t.Errorf("expected 64-char hex hash, got %d chars: %q", len(art.Hash), art.Hash)
	}
}

func TestBuilder_ArtifactIDFormat(t *testing.T) {
	b, _ := fakeBuilder()
	res := makeResult("macd_crossover", 99, []string{"MSFT"})

	art, err := b.BuildFromBacktest(context.Background(), "2.0.0",
		nil, res,
		domainArtifacts.RiskProfile{AllowedOrderTypes: []string{"LMT"}},
		"research-runtime",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(art.ArtifactID, "strat_macd_crossover_") {
		t.Errorf("ArtifactID %q does not have expected prefix", art.ArtifactID)
	}
}

func TestBuilder_DataWindowFromResult(t *testing.T) {
	b, _ := fakeBuilder()
	res := makeResult("rsi_momentum", 1, []string{"AAPL", "MSFT"})

	art, err := b.BuildFromBacktest(context.Background(), "1.0.0",
		nil, res,
		domainArtifacts.RiskProfile{AllowedOrderTypes: []string{"LMT"}},
		"test",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if art.DataWindow == nil {
		t.Fatal("DataWindow must not be nil")
	}
	if !art.DataWindow.From.Equal(res.StartDate) {
		t.Errorf("DataWindow.From mismatch: got %v, want %v", art.DataWindow.From, res.StartDate)
	}
	if !art.DataWindow.To.Equal(res.EndDate) {
		t.Errorf("DataWindow.To mismatch: got %v, want %v", art.DataWindow.To, res.EndDate)
	}
	if len(art.DataWindow.Symbols) != 2 {
		t.Errorf("expected 2 symbols, got %d", len(art.DataWindow.Symbols))
	}
}

// TestBuilder_SymbolFallback checks that result.Symbol is used when Symbols is empty.
func TestBuilder_SymbolFallback(t *testing.T) {
	b, _ := fakeBuilder()
	// Construct result with empty Symbols slice but non-empty Symbol field.
	res := makeResult("rsi_momentum", 2, []string{})
	res.BacktestResult.Symbol = "GOOGL"

	art, err := b.BuildFromBacktest(context.Background(), "1.0.0",
		nil, res,
		domainArtifacts.RiskProfile{AllowedOrderTypes: []string{"LMT"}},
		"test",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if art.DataWindow == nil || len(art.DataWindow.Symbols) != 1 {
		t.Fatalf("expected 1 symbol fallback from Symbol field, got: %v", art.DataWindow)
	}
	if art.DataWindow.Symbols[0] != "GOOGL" {
		t.Errorf("expected symbol GOOGL, got %q", art.DataWindow.Symbols[0])
	}
}

func TestBuilder_ValidationMetrics(t *testing.T) {
	b, _ := fakeBuilder()
	res := makeResult("rsi_momentum", 3, []string{"AAPL"})

	art, err := b.BuildFromBacktest(context.Background(), "1.0.0",
		nil, res,
		domainArtifacts.RiskProfile{AllowedOrderTypes: []string{"LMT"}},
		"test",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if art.Validation == nil {
		t.Fatal("Validation must not be nil")
	}

	required := []string{
		"total_trades", "winning_trades", "losing_trades", "win_rate",
		"total_return_pct", "max_drawdown", "sharpe_ratio", "profit_factor",
		"avg_r", "duration_ms",
	}
	for _, key := range required {
		if _, ok := art.Validation.Metrics[key]; !ok {
			t.Errorf("missing validation metric: %q", key)
		}
	}
}

// TestBuilder_DeterministicBacktestRunID checks that the same RunID always maps to
// the same BacktestRunID UUID (deterministic replay guarantee).
func TestBuilder_DeterministicBacktestRunID(t *testing.T) {
	b, _ := fakeBuilder()

	res1 := makeResult("rsi_momentum", 42, []string{"AAPL"})
	res2 := makeResult("rsi_momentum", 42, []string{"AAPL"})
	// Force identical RunIDs
	res2.RunID = res1.RunID

	art1, err := b.BuildFromBacktest(context.Background(), "1.0.0",
		nil, res1,
		domainArtifacts.RiskProfile{AllowedOrderTypes: []string{"LMT"}},
		"test",
	)
	if err != nil {
		t.Fatalf("build 1 failed: %v", err)
	}
	art2, err := b.BuildFromBacktest(context.Background(), "1.0.0",
		nil, res2,
		domainArtifacts.RiskProfile{AllowedOrderTypes: []string{"LMT"}},
		"test",
	)
	if err != nil {
		t.Fatalf("build 2 failed: %v", err)
	}

	if art1.Validation.BacktestRunID != art2.Validation.BacktestRunID {
		t.Errorf("same RunID should produce same BacktestRunID UUID: %v vs %v",
			art1.Validation.BacktestRunID, art2.Validation.BacktestRunID)
	}
}

// TestBuilder_DraftApprovalCreated checks the approval record persisted with the artifact.
func TestBuilder_DraftApprovalCreated(t *testing.T) {
	b, fs := fakeBuilder()
	res := makeResult("rsi_momentum", 5, []string{"AAPL"})

	_, err := b.BuildFromBacktest(context.Background(), "1.0.0",
		nil, res,
		domainArtifacts.RiskProfile{AllowedOrderTypes: []string{"LMT"}},
		"test-service",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(fs.approvals) != 1 {
		t.Fatalf("expected 1 approval record, got %d", len(fs.approvals))
	}
	ap := fs.approvals[0]
	if ap.State != domainArtifacts.StateDraft {
		t.Errorf("expected approval state DRAFT, got %q", ap.State)
	}
	if ap.ArtifactID == uuid.Nil {
		t.Error("approval ArtifactID must not be nil UUID")
	}
	if !ap.ValidationPassed {
		t.Error("ValidationPassed should be true for a freshly built artifact")
	}
	if !strings.Contains(ap.StateChangeReason, res.RunID) {
		t.Errorf("StateChangeReason should mention RunID %q, got %q", res.RunID, ap.StateChangeReason)
	}
}

// TestBuilder_ArtifactIDLinksToApproval verifies the saved artifact and approval share UUIDs.
func TestBuilder_ArtifactIDLinksToApproval(t *testing.T) {
	b, fs := fakeBuilder()
	res := makeResult("rsi_momentum", 6, []string{"AAPL"})

	art, err := b.BuildFromBacktest(context.Background(), "1.0.0",
		nil, res,
		domainArtifacts.RiskProfile{AllowedOrderTypes: []string{"LMT"}},
		"test",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(fs.artifacts) != 1 || len(fs.approvals) != 1 {
		t.Fatalf("expected 1 artifact and 1 approval, got %d/%d", len(fs.artifacts), len(fs.approvals))
	}
	if fs.approvals[0].ArtifactID != art.ID {
		t.Errorf("approval.ArtifactID %v != artifact.ID %v", fs.approvals[0].ArtifactID, art.ID)
	}
}

// TestBuilder_StoreArtifactError checks error propagation from CreateArtifact.
func TestBuilder_StoreArtifactError(t *testing.T) {
	fs := &fakeArtifactStore{failOn: "artifact"}
	b := NewBuilderWithStore(fs)
	res := makeResult("rsi_momentum", 7, []string{"AAPL"})

	_, err := b.BuildFromBacktest(context.Background(), "1.0.0",
		nil, res,
		domainArtifacts.RiskProfile{AllowedOrderTypes: []string{"LMT"}},
		"test",
	)
	if err == nil {
		t.Fatal("expected error from CreateArtifact, got nil")
	}
	if !strings.Contains(err.Error(), "failed to save artifact") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestBuilder_StoreApprovalError checks error propagation from CreateApproval.
func TestBuilder_StoreApprovalError(t *testing.T) {
	fs := &fakeArtifactStore{failOn: "approval"}
	b := NewBuilderWithStore(fs)
	res := makeResult("rsi_momentum", 8, []string{"AAPL"})

	_, err := b.BuildFromBacktest(context.Background(), "1.0.0",
		nil, res,
		domainArtifacts.RiskProfile{AllowedOrderTypes: []string{"LMT"}},
		"test",
	)
	if err == nil {
		t.Fatal("expected error from CreateApproval, got nil")
	}
	if !strings.Contains(err.Error(), "failed to create draft approval") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestBuilder_ParamsStoredInStrategy verifies strategy params are round-tripped into the artifact.
func TestBuilder_ParamsStoredInStrategy(t *testing.T) {
	b, _ := fakeBuilder()
	res := makeResult("rsi_momentum", 9, []string{"AAPL"})
	params := map[string]any{"rsi_period": 14, "entry": 30, "exit": 70}

	art, err := b.BuildFromBacktest(context.Background(), "2.1.0",
		params, res,
		domainArtifacts.RiskProfile{AllowedOrderTypes: []string{"LMT"}},
		"test",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if art.Strategy.Version != "2.1.0" {
		t.Errorf("expected version 2.1.0, got %q", art.Strategy.Version)
	}
	if art.Strategy.Params == nil {
		t.Fatal("strategy params must not be nil")
	}
	for k, want := range params {
		got, ok := art.Strategy.Params[k]
		if !ok {
			t.Errorf("missing param %q", k)
			continue
		}
		// JSON unmarshal will produce float64 for numbers; compare via Sprintf.
		if fmt.Sprintf("%v", got) != fmt.Sprintf("%v", want) {
			t.Errorf("param %q: got %v, want %v", k, got, want)
		}
	}
}

// TestBuilder_SchemaVersion verifies the schema version is always "1.0.0".
func TestBuilder_SchemaVersion(t *testing.T) {
	b, _ := fakeBuilder()
	res := makeResult("rsi_momentum", 10, []string{"AAPL"})

	art, err := b.BuildFromBacktest(context.Background(), "1.0.0",
		nil, res,
		domainArtifacts.RiskProfile{AllowedOrderTypes: []string{"LMT"}},
		"test",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if art.SchemaVersion != "1.0.0" {
		t.Errorf("expected SchemaVersion 1.0.0, got %q", art.SchemaVersion)
	}
}

// TestBuilder_CreatedByPreserved verifies the createdBy field is stored as-is.
func TestBuilder_CreatedByPreserved(t *testing.T) {
	b, _ := fakeBuilder()
	res := makeResult("rsi_momentum", 11, []string{"AAPL"})

	art, err := b.BuildFromBacktest(context.Background(), "1.0.0",
		nil, res,
		domainArtifacts.RiskProfile{AllowedOrderTypes: []string{"LMT"}},
		"alice@research",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if art.CreatedBy != "alice@research" {
		t.Errorf("expected CreatedBy alice@research, got %q", art.CreatedBy)
	}
}
