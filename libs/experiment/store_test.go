package experiment_test

import (
	"testing"
	"time"

	"jax-trading-assistant/libs/experiment"
)

// ─── helpers ──────────────────────────────────────────────────────────────────

func openStore(t *testing.T) *experiment.Store {
	t.Helper()
	s, err := experiment.Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return s
}

func createExp(t *testing.T, s *experiment.Store, name string) *experiment.Experiment {
	t.Helper()
	e, err := s.CreateExperiment(name, "test", []string{"unit"})
	if err != nil {
		t.Fatalf("CreateExperiment: %v", err)
	}
	return e
}

// ─── CreateExperiment ─────────────────────────────────────────────────────────

func TestCreateExperiment(t *testing.T) {
	s := openStore(t)
	e := createExp(t, s, "TestExp")

	if e.ID == "" {
		t.Error("expected non-empty ID")
	}
	if e.Name != "TestExp" {
		t.Errorf("Name: got %q want %q", e.Name, "TestExp")
	}
	if e.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}
}

func TestCreateExperimentDuplicateNameReturnsError(t *testing.T) {
	s := openStore(t)
	createExp(t, s, "dup")

	if _, err := s.CreateExperiment("dup", "", nil); err == nil {
		t.Fatal("expected error for duplicate name, got nil")
	}
}

func TestCreateExperimentEmptyNameReturnsError(t *testing.T) {
	s := openStore(t)
	if _, err := s.CreateExperiment("", "", nil); err == nil {
		t.Fatal("expected error for empty name, got nil")
	}
}

// ─── GetExperiment / GetByName / List / Delete ────────────────────────────────

func TestGetExperiment(t *testing.T) {
	s := openStore(t)
	e := createExp(t, s, "GE")

	got, err := s.GetExperiment(e.ID)
	if err != nil {
		t.Fatalf("GetExperiment: %v", err)
	}
	if got.ID != e.ID {
		t.Errorf("ID mismatch")
	}
}

func TestGetExperimentNotFoundReturnsError(t *testing.T) {
	s := openStore(t)
	if _, err := s.GetExperiment("missing"); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetExperimentByName(t *testing.T) {
	s := openStore(t)
	e := createExp(t, s, "ByName")
	got, err := s.GetExperimentByName("ByName")
	if err != nil {
		t.Fatalf("GetExperimentByName: %v", err)
	}
	if got.ID != e.ID {
		t.Errorf("ID mismatch")
	}
}

func TestListExperiments(t *testing.T) {
	s := openStore(t)
	createExp(t, s, "A")
	createExp(t, s, "B")
	createExp(t, s, "C")

	list := s.ListExperiments()
	if len(list) != 3 {
		t.Fatalf("List: got %d, want 3", len(list))
	}
}

func TestDeleteExperiment(t *testing.T) {
	s := openStore(t)
	e := createExp(t, s, "DEL")

	if err := s.DeleteExperiment(e.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.GetExperiment(e.ID); err == nil {
		t.Fatal("expected error after delete, got nil")
	}
}

// ─── Run lifecycle ────────────────────────────────────────────────────────────

func TestStartCompleteRun(t *testing.T) {
	s := openStore(t)
	e := createExp(t, s, "Runs")

	params := experiment.RunParams{
		StrategyID: "rsi_v1",
		DatasetID:  "ds-123",
		Seed:       42,
	}

	run, err := s.StartRun(e.ID, "run-1", params)
	if err != nil {
		t.Fatalf("StartRun: %v", err)
	}
	if run.Status != experiment.StatusRunning {
		t.Errorf("Status: got %q want %q", run.Status, experiment.StatusRunning)
	}

	metrics := experiment.RunMetrics{
		TotalTrades: 10,
		WinRate:     0.6,
		SharpeRatio: 1.2,
	}
	if err := s.CompleteRun(run.ID, metrics, 500); err != nil {
		t.Fatalf("CompleteRun: %v", err)
	}

	got, _, err := s.GetRun(run.ID)
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if got.Status != experiment.StatusCompleted {
		t.Errorf("Status after complete: got %q want %q", got.Status, experiment.StatusCompleted)
	}
	if got.Metrics.TotalTrades != 10 {
		t.Errorf("TotalTrades: got %d want 10", got.Metrics.TotalTrades)
	}
	if got.CompletedAt == nil {
		t.Error("expected CompletedAt to be set")
	}
}

func TestFailRun(t *testing.T) {
	s := openStore(t)
	e := createExp(t, s, "FailCase")

	run, _ := s.StartRun(e.ID, "", experiment.RunParams{StrategyID: "x"})
	if err := s.FailRun(run.ID, "timeout"); err != nil {
		t.Fatalf("FailRun: %v", err)
	}

	got, _, _ := s.GetRun(run.ID)
	if got.Status != experiment.StatusFailed {
		t.Errorf("Status: got %q want %q", got.Status, experiment.StatusFailed)
	}
	if got.ErrorMessage != "timeout" {
		t.Errorf("ErrorMessage: %q", got.ErrorMessage)
	}
}

func TestListRuns(t *testing.T) {
	s := openStore(t)
	e := createExp(t, s, "ListR")

	s.StartRun(e.ID, "r1", experiment.RunParams{}) //nolint:errcheck
	s.StartRun(e.ID, "r2", experiment.RunParams{}) //nolint:errcheck

	runs, err := s.ListRuns(e.ID)
	if err != nil {
		t.Fatalf("ListRuns: %v", err)
	}
	if len(runs) != 2 {
		t.Errorf("ListRuns: got %d, want 2", len(runs))
	}
}

// ─── BestRun ─────────────────────────────────────────────────────────────────

func TestBestRun(t *testing.T) {
	s := openStore(t)
	e := createExp(t, s, "BestR")

	r1, _ := s.StartRun(e.ID, "low", experiment.RunParams{})
	r2, _ := s.StartRun(e.ID, "high", experiment.RunParams{})

	s.CompleteRun(r1.ID, experiment.RunMetrics{SharpeRatio: 0.5}, 100) //nolint:errcheck
	s.CompleteRun(r2.ID, experiment.RunMetrics{SharpeRatio: 1.8}, 100) //nolint:errcheck

	best, err := s.BestRun(e.ID)
	if err != nil {
		t.Fatalf("BestRun: %v", err)
	}
	if best.ID != r2.ID {
		t.Errorf("BestRun ID: got %s want %s", best.ID, r2.ID)
	}
}

func TestBestRunNoCompletedRunsReturnsError(t *testing.T) {
	s := openStore(t)
	e := createExp(t, s, "NoCmp")
	s.StartRun(e.ID, "", experiment.RunParams{}) //nolint:errcheck

	if _, err := s.BestRun(e.ID); err == nil {
		t.Fatal("expected error when no completed runs, got nil")
	}
}

// ─── Persistence ─────────────────────────────────────────────────────────────

func TestPersistence(t *testing.T) {
	dir := t.TempDir()

	s1, _ := experiment.Open(dir)
	e, _ := s1.CreateExperiment("Persist", "test store survives reopen", nil)
	run, _ := s1.StartRun(e.ID, "r1", experiment.RunParams{Seed: 99})
	s1.CompleteRun(run.ID, experiment.RunMetrics{SharpeRatio: 0.9}, 200) //nolint:errcheck

	s2, err := experiment.Open(dir)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}

	got, err := s2.GetExperiment(e.ID)
	if err != nil {
		t.Fatalf("GetExperiment after reopen: %v", err)
	}
	if len(got.Runs) != 1 {
		t.Fatalf("runs after reopen: got %d, want 1", len(got.Runs))
	}
	if got.Runs[0].Params.Seed != 99 {
		t.Errorf("Seed preserved: got %d want 99", got.Runs[0].Params.Seed)
	}
}

// ─── ParamHash ────────────────────────────────────────────────────────────────

func TestParamHashDeterministic(t *testing.T) {
	p := experiment.RunParams{StrategyID: "x", DatasetID: "y", Seed: 1}
	h1 := p.ParamHash()
	h2 := p.ParamHash()
	if h1 != h2 {
		t.Errorf("ParamHash not deterministic: %s != %s", h1, h2)
	}
	if len(h1) != 12 {
		t.Errorf("ParamHash length: got %d want 12", len(h1))
	}
}

func TestParamHashDistinct(t *testing.T) {
	p1 := experiment.RunParams{Seed: 1}
	p2 := experiment.RunParams{Seed: 2}
	if p1.ParamHash() == p2.ParamHash() {
		t.Error("expected different hashes for different seeds")
	}
}

// Keep time import used — example shows time used later when more tests are added
var _ = time.Now
