// Package experiment implements L06: experiment tracking for backtest and
// walk-forward validation runs.
//
// An Experiment groups related runs under a single named investigation (e.g.
// "RSI parameter sweep Q1-2024").  Each Run records its parameters, result
// metrics, and the dataset + strategy IDs so that any run can be reproduced
// exactly by feeding those IDs back into the backtest engine.
//
// Experiments and their Runs are persisted as a JSON file on disk (one file
// per experiment store directory).
package experiment

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ─── Schema ───────────────────────────────────────────────────────────────────

const storeFile = "experiments.json"

// ─── Status ───────────────────────────────────────────────────────────────────

// Status represents the lifecycle state of a Run.
type Status string

const (
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
)

// ─── RunParams ────────────────────────────────────────────────────────────────

// RunParams captures all inputs to a backtest or walk-forward run so it can
// be reproduced.  All fields are optional; callers should fill what they know.
type RunParams struct {
	// StrategyID as registered in libs/strategies.Registry.
	StrategyID string `json:"strategy_id"`
	// DatasetID from libs/dataset.Registry.
	DatasetID string `json:"dataset_id"`
	// DatasetHash is the SHA-256 hex digest at the time of the run.
	// Presence allows detecting file drift before replay.
	DatasetHash string `json:"dataset_hash,omitempty"`
	// Seed for deterministic replay.
	Seed int64 `json:"seed"`
	// StartDate / EndDate in YYYY-MM-DD format.
	StartDate string `json:"start_date,omitempty"`
	EndDate   string `json:"end_date,omitempty"`
	// InitialCapital in USD.
	InitialCapital float64 `json:"initial_capital,omitempty"`
	// RiskPerTrade fraction.
	RiskPerTrade float64 `json:"risk_per_trade,omitempty"`
	// Extra holds any additional key/value metadata (e.g. custom hyperparams).
	Extra map[string]any `json:"extra,omitempty"`
}

// ParamHash returns a deterministic 12-char SHA-256 prefix over the
// serialised RunParams.  Use this for deduplication.
func (p RunParams) ParamHash() string {
	b, _ := json.Marshal(p)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])[:12]
}

// ─── RunMetrics ───────────────────────────────────────────────────────────────

// RunMetrics holds the measured output of a single run.
type RunMetrics struct {
	TotalTrades    int     `json:"total_trades"`
	WinningTrades  int     `json:"winning_trades"`
	LosingTrades   int     `json:"losing_trades"`
	WinRate        float64 `json:"win_rate"`
	TotalReturn    float64 `json:"total_return"`
	AnnualisedRet  float64 `json:"annualised_return,omitempty"`
	MaxDrawdown    float64 `json:"max_drawdown"`
	SharpeRatio    float64 `json:"sharpe_ratio"`
	FinalCapital   float64 `json:"final_capital,omitempty"`
	// WalkForward fields (populated for WF runs)
	WFER           float64 `json:"wfer,omitempty"`
	PassRate       float64 `json:"pass_rate,omitempty"`
	StabilityScore float64 `json:"stability_score,omitempty"`
	OOSWindows     int     `json:"oos_windows,omitempty"`
}

// ─── Run ──────────────────────────────────────────────────────────────────────

// Run records a single backtest or walk-forward execution within an Experiment.
type Run struct {
	// ID is the run UUID.
	ID string `json:"id"`
	// ExperimentID is the parent experiment UUID.
	ExperimentID string `json:"experiment_id"`
	// Name is an optional human-readable label (e.g. "RSI70 run #3").
	Name string `json:"name,omitempty"`
	// Status tracks the run lifecycle.
	Status Status `json:"status"`
	// Params are the inputs used for this run.
	Params RunParams `json:"params"`
	// Metrics are the measured outputs (empty while running).
	Metrics RunMetrics `json:"metrics,omitempty"`
	// ErrorMessage is populated on StatusFailed.
	ErrorMessage string `json:"error,omitempty"`
	// DurationMs is how long the run took in milliseconds.
	DurationMs int64 `json:"duration_ms,omitempty"`
	// StartedAt / CompletedAt are wall-clock timestamps.
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// ─── Experiment ───────────────────────────────────────────────────────────────

// Experiment groups related runs under one investigation.
type Experiment struct {
	// ID is a UUID assigned at creation.
	ID string `json:"id"`
	// Name is a unique human-readable name.
	Name string `json:"name"`
	// Description is an optional free-form note.
	Description string `json:"description,omitempty"`
	// Tags are searchable labels.
	Tags []string `json:"tags,omitempty"`
	// CreatedAt is when CreateExperiment was called.
	CreatedAt time.Time `json:"created_at"`
	// Runs contains all run records for this experiment.
	Runs []Run `json:"runs"`
}

// ─── Store ────────────────────────────────────────────────────────────────────

// Store is a thread-safe persistent store of Experiments and their Runs.
type Store struct {
	mu          sync.RWMutex
	dir         string
	experiments map[string]*Experiment // keyed by ID
}

// Open loads (or creates) a Store backed by dir.
func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("experiment.Open: mkdir: %w", err)
	}

	s := &Store{
		dir:         dir,
		experiments: make(map[string]*Experiment),
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

// CreateExperiment adds a new experiment to the store.
// Names must be unique across all experiments in the store.
func (s *Store) CreateExperiment(name, description string, tags []string) (*Experiment, error) {
	if name == "" {
		return nil, fmt.Errorf("experiment.CreateExperiment: name must not be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, e := range s.experiments {
		if e.Name == name {
			return nil, fmt.Errorf("experiment.CreateExperiment: name %q already exists (id=%s)", name, e.ID)
		}
	}

	exp := &Experiment{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		Tags:        tags,
		CreatedAt:   time.Now().UTC(),
		Runs:        []Run{},
	}
	s.experiments[exp.ID] = exp

	if err := s.save(); err != nil {
		delete(s.experiments, exp.ID)
		return nil, fmt.Errorf("experiment.CreateExperiment: persist: %w", err)
	}

	log.Printf("[experiment] created name=%q id=%s", name, exp.ID[:8])
	return copyExperiment(exp), nil
}

// GetExperiment returns the Experiment with the given ID.
func (s *Store) GetExperiment(id string) (*Experiment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	e, ok := s.experiments[id]
	if !ok {
		return nil, fmt.Errorf("experiment.GetExperiment: id %q not found", id)
	}
	return copyExperiment(e), nil
}

// GetExperimentByName returns the experiment with the given Name.
func (s *Store) GetExperimentByName(name string) (*Experiment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, e := range s.experiments {
		if e.Name == name {
			return copyExperiment(e), nil
		}
	}
	return nil, fmt.Errorf("experiment.GetExperimentByName: %q not found", name)
}

// ListExperiments returns all experiments sorted by CreatedAt ascending.
func (s *Store) ListExperiments() []Experiment {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]Experiment, 0, len(s.experiments))
	for _, e := range s.experiments {
		out = append(out, *copyExperiment(e))
	}
	slices.SortFunc(out, func(a, b Experiment) int {
		return a.CreatedAt.Compare(b.CreatedAt)
	})
	return out
}

// DeleteExperiment removes an experiment and all its runs.
func (s *Store) DeleteExperiment(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.experiments[id]; !ok {
		return fmt.Errorf("experiment.DeleteExperiment: id %q not found", id)
	}
	delete(s.experiments, id)
	return s.save()
}

// ─── Run management ───────────────────────────────────────────────────────────

// StartRun records a run in StatusRunning state.
// Returns the Run (with a freshly assigned UUID) for later completion via
// CompleteRun or FailRun.
func (s *Store) StartRun(experimentID string, name string, params RunParams) (*Run, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	exp, ok := s.experiments[experimentID]
	if !ok {
		return nil, fmt.Errorf("experiment.StartRun: experiment %q not found", experimentID)
	}

	run := Run{
		ID:           uuid.New().String(),
		ExperimentID: experimentID,
		Name:         name,
		Status:       StatusRunning,
		Params:       params,
		StartedAt:    time.Now().UTC(),
	}
	exp.Runs = append(exp.Runs, run)

	if err := s.save(); err != nil {
		exp.Runs = exp.Runs[:len(exp.Runs)-1]
		return nil, fmt.Errorf("experiment.StartRun: persist: %w", err)
	}

	runCopy := run
	log.Printf("[experiment] run started exp=%s run=%s params_hash=%s",
		experimentID[:8], run.ID[:8], params.ParamHash())
	return &runCopy, nil
}

// CompleteRun marks a run as completed with the given metrics.
func (s *Store) CompleteRun(runID string, metrics RunMetrics, durationMs int64) error {
	return s.updateRun(runID, func(r *Run) {
		now := time.Now().UTC()
		r.Status = StatusCompleted
		r.Metrics = metrics
		r.DurationMs = durationMs
		r.CompletedAt = &now
		log.Printf("[experiment] run completed run=%s trades=%d winRate=%.1f%%",
			runID[:8], metrics.TotalTrades, metrics.WinRate*100)
	})
}

// FailRun marks a run as failed with an error message.
func (s *Store) FailRun(runID string, errMsg string) error {
	return s.updateRun(runID, func(r *Run) {
		now := time.Now().UTC()
		r.Status = StatusFailed
		r.ErrorMessage = errMsg
		r.CompletedAt = &now
		log.Printf("[experiment] run failed run=%s error=%q", runID[:8], errMsg)
	})
}

// GetRun returns the Run with the given ID, searching across all experiments.
func (s *Store) GetRun(runID string) (*Run, string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, exp := range s.experiments {
		for i := range exp.Runs {
			if exp.Runs[i].ID == runID {
				r := exp.Runs[i]
				return &r, exp.ID, nil
			}
		}
	}
	return nil, "", fmt.Errorf("experiment.GetRun: run %q not found", runID)
}

// ListRuns returns all runs for the given experiment ID, sorted by StartedAt.
func (s *Store) ListRuns(experimentID string) ([]Run, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	exp, ok := s.experiments[experimentID]
	if !ok {
		return nil, fmt.Errorf("experiment.ListRuns: experiment %q not found", experimentID)
	}

	runs := make([]Run, len(exp.Runs))
	copy(runs, exp.Runs)
	slices.SortFunc(runs, func(a, b Run) int {
		return a.StartedAt.Compare(b.StartedAt)
	})
	return runs, nil
}

// BestRun returns the completed Run with the highest Sharpe ratio in an
// experiment.  Returns an error if no completed runs exist.
func (s *Store) BestRun(experimentID string) (*Run, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	exp, ok := s.experiments[experimentID]
	if !ok {
		return nil, fmt.Errorf("experiment.BestRun: experiment %q not found", experimentID)
	}

	var best *Run
	for i := range exp.Runs {
		r := &exp.Runs[i]
		if r.Status != StatusCompleted {
			continue
		}
		if best == nil || r.Metrics.SharpeRatio > best.Metrics.SharpeRatio {
			best = r
		}
	}
	if best == nil {
		return nil, fmt.Errorf("experiment.BestRun: no completed runs in experiment %q", experimentID)
	}
	runCopy := *best
	return &runCopy, nil
}

// ─── internals ────────────────────────────────────────────────────────────────

func (s *Store) updateRun(runID string, mutate func(*Run)) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, exp := range s.experiments {
		for i := range exp.Runs {
			if exp.Runs[i].ID == runID {
				mutate(&exp.Runs[i])
				return s.save()
			}
		}
	}
	return fmt.Errorf("experiment.updateRun: run %q not found", runID)
}

func (s *Store) storePath() string {
	return filepath.Join(s.dir, storeFile)
}

type storeSchema struct {
	Experiments []*Experiment `json:"experiments"`
}

func (s *Store) load() error {
	f, err := os.Open(s.storePath())
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("experiment: open store: %w", err)
	}
	defer f.Close()

	var schema storeSchema
	if err := json.NewDecoder(f).Decode(&schema); err != nil {
		return fmt.Errorf("experiment: decode store: %w", err)
	}
	for _, e := range schema.Experiments {
		s.experiments[e.ID] = e
	}
	return nil
}

func (s *Store) save() error {
	exps := make([]*Experiment, 0, len(s.experiments))
	for _, e := range s.experiments {
		exps = append(exps, e)
	}
	slices.SortFunc(exps, func(a, b *Experiment) int {
		return a.CreatedAt.Compare(b.CreatedAt)
	})

	schema := storeSchema{Experiments: exps}
	tmp := s.storePath() + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("experiment: create store tmp: %w", err)
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(schema); err != nil {
		f.Close()
		os.Remove(tmp)
		return fmt.Errorf("experiment: encode store: %w", err)
	}
	f.Close()
	if err := os.Rename(tmp, s.storePath()); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("experiment: rename store: %w", err)
	}
	return nil
}

// copyExperiment returns a deep-enough copy of an Experiment (slice header
// is copied; individual Run values are immutable value types).
func copyExperiment(e *Experiment) *Experiment {
	cp := *e
	cp.Runs = make([]Run, len(e.Runs))
	copy(cp.Runs, e.Runs)
	return &cp
}
