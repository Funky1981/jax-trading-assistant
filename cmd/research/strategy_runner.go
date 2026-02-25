package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type ResearchProjectRunRequest struct {
	ProjectID        string         `json:"projectId"`
	InstanceID       string         `json:"instanceId"`
	StrategyID       string         `json:"strategyId"`
	StrategyConfigID string         `json:"strategyConfigId"`
	DatasetID        string         `json:"datasetId"`
	SymbolsOverride  []string       `json:"symbolsOverride"`
	From             string         `json:"from"`
	To               string         `json:"to"`
	TrainFrom        string         `json:"trainFrom"`
	TrainTo          string         `json:"trainTo"`
	TestFrom         string         `json:"testFrom"`
	TestTo           string         `json:"testTo"`
	ParameterGrid    map[string]any `json:"parameterGrid"`
	Seed             int64          `json:"seed"`
	InitialCapital   float64        `json:"initialCapital"`
	RiskPerTrade     float64        `json:"riskPerTrade"`
	MaxCombos        int            `json:"maxCombos"`
	Async            bool           `json:"async"`
}

type ResearchProjectRunEntry struct {
	Index     int            `json:"index"`
	Combo     map[string]any `json:"combo"`
	TrainRun  map[string]any `json:"train"`
	TestRun   map[string]any `json:"test"`
	RankScore float64        `json:"rankScore"`
	Error     string         `json:"error,omitempty"`
}

type ResearchProjectRunResponse struct {
	JobID        string                    `json:"jobId"`
	ProjectID    string                    `json:"projectId"`
	Status       string                    `json:"status"`
	TotalCombos  int                       `json:"totalCombos"`
	FailedCombos int                       `json:"failedCombos"`
	Runs         []ResearchProjectRunEntry `json:"runs"`
	StartedAt    time.Time                 `json:"startedAt"`
	CompletedAt  *time.Time                `json:"completedAt,omitempty"`
	Error        string                    `json:"error,omitempty"`
}

type researchRunnerJob struct {
	id        string
	req       ResearchProjectRunRequest
	status    string
	result    ResearchProjectRunResponse
	err       string
	startedAt time.Time
	done      chan struct{}
}

type researchRunnerManager struct {
	deps    *backtestDeps
	workers int
	queue   chan string
	stopCh  chan struct{}
	wg      sync.WaitGroup

	mu   sync.RWMutex
	jobs map[string]*researchRunnerJob
}

func newResearchRunnerManager(deps *backtestDeps, workers int) *researchRunnerManager {
	if workers <= 0 {
		workers = 2
	}
	m := &researchRunnerManager{
		deps:    deps,
		workers: workers,
		queue:   make(chan string, 256),
		stopCh:  make(chan struct{}),
		jobs:    make(map[string]*researchRunnerJob),
	}
	for i := 0; i < workers; i++ {
		m.wg.Add(1)
		go m.workerLoop(i + 1)
	}
	return m
}

func (m *researchRunnerManager) Stop() {
	close(m.stopCh)
	m.wg.Wait()
}

func (m *researchRunnerManager) Submit(req ResearchProjectRunRequest) string {
	jobID := uuid.NewString()
	job := &researchRunnerJob{
		id:     jobID,
		req:    req,
		status: "queued",
		done:   make(chan struct{}),
	}
	m.mu.Lock()
	m.jobs[jobID] = job
	m.mu.Unlock()
	m.queue <- jobID
	return jobID
}

func (m *researchRunnerManager) Wait(ctx context.Context, jobID string) (*ResearchProjectRunResponse, error) {
	job, ok := m.getJob(jobID)
	if !ok {
		return nil, fmt.Errorf("job not found")
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-job.done:
		resp := m.publicJob(job)
		return &resp, nil
	}
}

func (m *researchRunnerManager) Get(jobID string) (*ResearchProjectRunResponse, bool) {
	job, ok := m.getJob(jobID)
	if !ok {
		return nil, false
	}
	resp := m.publicJob(job)
	return &resp, true
}

func (m *researchRunnerManager) getJob(jobID string) (*researchRunnerJob, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	job, ok := m.jobs[jobID]
	return job, ok
}

func (m *researchRunnerManager) publicJob(job *researchRunnerJob) ResearchProjectRunResponse {
	m.mu.RLock()
	defer m.mu.RUnlock()
	resp := job.result
	resp.JobID = job.id
	resp.ProjectID = job.req.ProjectID
	resp.Status = job.status
	if job.err != "" && resp.Error == "" {
		resp.Error = job.err
	}
	if resp.StartedAt.IsZero() {
		resp.StartedAt = job.startedAt
	}
	return resp
}

func (m *researchRunnerManager) workerLoop(workerID int) {
	defer m.wg.Done()
	for {
		select {
		case <-m.stopCh:
			return
		case jobID := <-m.queue:
			m.executeJob(workerID, jobID)
		}
	}
}

func (m *researchRunnerManager) executeJob(workerID int, jobID string) {
	m.mu.Lock()
	job, ok := m.jobs[jobID]
	if !ok {
		m.mu.Unlock()
		return
	}
	job.status = "running"
	job.startedAt = time.Now().UTC()
	m.mu.Unlock()

	log.Printf("[research-runner] worker=%d job=%s project=%s start", workerID, jobID, job.req.ProjectID)
	resp, err := runProjectSweep(context.Background(), m.deps, job.id, job.req)

	m.mu.Lock()
	if err != nil {
		job.status = "failed"
		job.err = err.Error()
		job.result = ResearchProjectRunResponse{
			ProjectID:    job.req.ProjectID,
			Status:       "failed",
			TotalCombos:  0,
			FailedCombos: 0,
			Runs:         nil,
			StartedAt:    job.startedAt,
			Error:        err.Error(),
		}
	} else {
		job.status = resp.Status
		job.result = resp
		job.result.ProjectID = job.req.ProjectID
		job.result.JobID = job.id
	}
	if job.result.CompletedAt == nil {
		now := time.Now().UTC()
		job.result.CompletedAt = &now
	}
	select {
	case <-job.done:
	default:
		close(job.done)
	}
	m.mu.Unlock()
	log.Printf("[research-runner] worker=%d job=%s status=%s failed=%d/%d", workerID, jobID, job.status, job.result.FailedCombos, job.result.TotalCombos)
}

func handleResearchProjectRun(mgr *researchRunnerManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req ResearchProjectRunRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		if req.ProjectID == "" {
			http.Error(w, "projectId is required", http.StatusBadRequest)
			return
		}
		jobID := mgr.Submit(req)
		if req.Async {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jobId":     jobID,
				"projectId": req.ProjectID,
				"status":    "queued",
			})
			return
		}
		timeout := 10 * time.Minute
		if raw := strings.TrimSpace(r.URL.Query().Get("timeoutSeconds")); raw != "" {
			if n, err := strconv.Atoi(raw); err == nil && n > 0 {
				timeout = time.Duration(n) * time.Second
			}
		}
		waitCtx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()
		resp, err := mgr.Wait(waitCtx, jobID)
		if err != nil {
			http.Error(w, fmt.Sprintf("job wait failed: %v", err), http.StatusGatewayTimeout)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func handleResearchProjectRunStatus(mgr *researchRunnerManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/research/projects/runs/"), "/")
		if id == "" {
			http.NotFound(w, r)
			return
		}
		resp, ok := mgr.Get(id)
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func runProjectSweep(ctx context.Context, deps *backtestDeps, jobID string, req ResearchProjectRunRequest) (ResearchProjectRunResponse, error) {
	symbols := req.SymbolsOverride
	if len(symbols) == 0 {
		symbols = []string{envOrDefault("BACKTEST_DEFAULT_SYMBOL", "SPY")}
	}
	datasetID := strings.TrimSpace(req.DatasetID)
	if datasetID == "" {
		datasetID = envOrDefault("BACKTEST_DATASET_ID", "")
	}
	if datasetID == "" {
		return ResearchProjectRunResponse{}, fmt.Errorf("datasetId is required")
	}
	strategyID := strings.TrimSpace(req.StrategyID)
	if strategyID == "" {
		strategyID = strings.TrimSpace(req.StrategyConfigID)
	}
	if strategyID == "" {
		strategyID = "rsi_momentum_v1"
	}
	maxCombos := req.MaxCombos
	if maxCombos <= 0 {
		maxCombos = 128
	}
	combos := runnerExpandParameterGrid(req.ParameterGrid, maxCombos)
	if len(combos) == 0 {
		combos = []map[string]any{{}}
	}
	trainFrom, trainTo, testFrom, testTo := runnerWalkForwardWindows(req)

	out := ResearchProjectRunResponse{
		JobID:        jobID,
		ProjectID:    req.ProjectID,
		Status:       "completed",
		TotalCombos:  len(combos),
		FailedCombos: 0,
		Runs:         make([]ResearchProjectRunEntry, 0, len(combos)),
		StartedAt:    time.Now().UTC(),
	}
	for idx, combo := range combos {
		seed := req.Seed
		if seed == 0 {
			seed = runnerDeterministicSeed(req.ProjectID, idx, combo)
		}
		btReqTrain := runnerApplyCombo(BacktestRequest{
			Strategy:       strategyID,
			Symbols:        symbols,
			StartDate:      trainFrom.Format(dateFmt),
			EndDate:        trainTo.Format(dateFmt),
			InitialCapital: req.InitialCapital,
			RiskPerTrade:   req.RiskPerTrade,
			DatasetID:      datasetID,
			Seed:           seed,
		}, combo)
		btReqTest := runnerApplyCombo(BacktestRequest{
			Strategy:       strategyID,
			Symbols:        symbols,
			StartDate:      testFrom.Format(dateFmt),
			EndDate:        testTo.Format(dateFmt),
			InitialCapital: req.InitialCapital,
			RiskPerTrade:   req.RiskPerTrade,
			DatasetID:      datasetID,
			Seed:           seed + 1,
		}, combo)
		trainResp, errTrain := runBacktest(ctx, deps, btReqTrain)
		testResp, errTest := runBacktest(ctx, deps, btReqTest)
		entry := ResearchProjectRunEntry{
			Index: idx,
			Combo: combo,
		}
		if errTrain != nil || errTest != nil {
			out.FailedCombos++
			entry.Error = strings.TrimSpace(errString(errTrain) + " " + errString(errTest))
			out.Runs = append(out.Runs, entry)
			continue
		}
		entry.TrainRun = map[string]any{
			"runId":   trainResp.RunID,
			"summary": trainResp,
		}
		entry.TestRun = map[string]any{
			"runId":   testResp.RunID,
			"summary": testResp,
		}
		entry.RankScore = runnerRankScore(testResp)
		out.Runs = append(out.Runs, entry)
	}
	if out.FailedCombos > 0 {
		out.Status = "degraded"
	}
	if out.FailedCombos == out.TotalCombos {
		out.Status = "failed"
		out.Error = "all parameter combinations failed"
	}
	now := time.Now().UTC()
	out.CompletedAt = &now
	return out, nil
}

func runnerWalkForwardWindows(req ResearchProjectRunRequest) (time.Time, time.Time, time.Time, time.Time) {
	fallbackFrom := runnerParseDate(req.From, time.Now().UTC().AddDate(0, 0, -30))
	fallbackTo := runnerParseDate(req.To, time.Now().UTC())
	if !fallbackTo.After(fallbackFrom) {
		fallbackTo = fallbackFrom.AddDate(0, 0, 5)
	}
	trainFrom := runnerParseDate(req.TrainFrom, fallbackFrom)
	trainTo := runnerParseDate(req.TrainTo, fallbackTo)
	testFrom := runnerParseDate(req.TestFrom, fallbackFrom)
	testTo := runnerParseDate(req.TestTo, fallbackTo)
	if strings.TrimSpace(req.TrainFrom) == "" || strings.TrimSpace(req.TrainTo) == "" || strings.TrimSpace(req.TestFrom) == "" || strings.TrimSpace(req.TestTo) == "" {
		split := fallbackFrom.Add(time.Duration(float64(fallbackTo.Sub(fallbackFrom)) * 0.7))
		trainFrom = fallbackFrom
		trainTo = split
		testFrom = split.Add(24 * time.Hour)
		testTo = fallbackTo
	}
	if !trainTo.After(trainFrom) {
		trainTo = trainFrom.AddDate(0, 0, 2)
	}
	if !testTo.After(testFrom) {
		testTo = testFrom.AddDate(0, 0, 2)
	}
	return trainFrom, trainTo, testFrom, testTo
}

func runnerParseDate(raw string, fallback time.Time) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback.UTC()
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t.UTC()
	}
	if t, err := time.Parse(dateFmt, raw); err == nil {
		return t.UTC()
	}
	return fallback.UTC()
}

func runnerExpandParameterGrid(grid map[string]any, maxCombos int) []map[string]any {
	if len(grid) == 0 {
		return nil
	}
	keys := make([]string, 0, len(grid))
	for k := range grid {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	valuesByKey := make([][]any, 0, len(keys))
	for _, key := range keys {
		switch typed := grid[key].(type) {
		case []any:
			if len(typed) == 0 {
				valuesByKey = append(valuesByKey, []any{nil})
			} else {
				valuesByKey = append(valuesByKey, typed)
			}
		default:
			valuesByKey = append(valuesByKey, []any{typed})
		}
	}
	out := make([]map[string]any, 0, 16)
	var walk func(i int, current map[string]any)
	walk = func(i int, current map[string]any) {
		if maxCombos > 0 && len(out) >= maxCombos {
			return
		}
		if i >= len(keys) {
			cloned := make(map[string]any, len(current))
			for k, v := range current {
				cloned[k] = v
			}
			out = append(out, cloned)
			return
		}
		key := keys[i]
		for _, value := range valuesByKey[i] {
			current[key] = value
			walk(i+1, current)
		}
		delete(current, key)
	}
	walk(0, map[string]any{})
	return out
}

func runnerApplyCombo(req BacktestRequest, combo map[string]any) BacktestRequest {
	out := req
	if out.Parameters == nil {
		out.Parameters = map[string]any{}
	}
	for key, value := range combo {
		switch key {
		case "strategyId", "strategy", "strategyConfigId":
			if s := strings.TrimSpace(fmt.Sprintf("%v", value)); s != "" {
				out.Strategy = s
			}
		case "datasetId":
			if s := strings.TrimSpace(fmt.Sprintf("%v", value)); s != "" {
				out.DatasetID = s
			}
		case "seed":
			if v, ok := runnerInt64(value); ok {
				out.Seed = v
			}
		case "initialCapital":
			if v, ok := runnerFloat64(value); ok {
				out.InitialCapital = v
			}
		case "riskPerTrade":
			if v, ok := runnerFloat64(value); ok {
				out.RiskPerTrade = v
			}
		case "symbols", "symbolsOverride":
			if vv := runnerStringSlice(value); len(vv) > 0 {
				out.Symbols = vv
			}
		case "parameters":
			if paramMap, ok := value.(map[string]any); ok {
				for k, v := range paramMap {
					out.Parameters[k] = v
				}
			}
		case "sessionTimezone":
			if s := strings.TrimSpace(fmt.Sprintf("%v", value)); s != "" {
				out.SessionTimezone = s
			}
		case "flattenByCloseTime":
			if s := strings.TrimSpace(fmt.Sprintf("%v", value)); s != "" {
				out.FlattenByCloseTime = s
			}
		default:
			out.Parameters[key] = value
		}
	}
	return out
}

func runnerRankScore(resp BacktestResponse) float64 {
	periodDays := float64(max(1, resp.TotalTrades/3))
	avgDaily := resp.TotalReturn / periodDays
	tailLoss := 0.0
	if resp.TotalReturn < 0 {
		tailLoss = -resp.TotalReturn
	}
	return (avgDaily * 100) - (resp.MaxDrawdown * 100) - (tailLoss * 50)
}

func runnerDeterministicSeed(projectID string, idx int, combo map[string]any) int64 {
	encoded, _ := json.Marshal(combo)
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%d|%s", projectID, idx, string(encoded))))
	return int64(sum[0])<<56 | int64(sum[1])<<48 | int64(sum[2])<<40 | int64(sum[3])<<32 |
		int64(sum[4])<<24 | int64(sum[5])<<16 | int64(sum[6])<<8 | int64(sum[7])
}

func runnerFloat64(v any) (float64, bool) {
	switch typed := v.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case json.Number:
		f, err := typed.Float64()
		return f, err == nil
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		return f, err == nil
	default:
		return 0, false
	}
}

func runnerInt64(v any) (int64, bool) {
	switch typed := v.(type) {
	case int64:
		return typed, true
	case int:
		return int64(typed), true
	case float64:
		return int64(typed), true
	case string:
		n, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		return n, err == nil
	default:
		return 0, false
	}
}

func runnerStringSlice(v any) []string {
	switch typed := v.(type) {
	case []string:
		out := make([]string, 0, len(typed))
		for _, s := range typed {
			if ss := strings.ToUpper(strings.TrimSpace(s)); ss != "" {
				out = append(out, ss)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(typed))
		for _, raw := range typed {
			ss := strings.ToUpper(strings.TrimSpace(fmt.Sprintf("%v", raw)))
			if ss != "" {
				out = append(out, ss)
			}
		}
		return out
	case string:
		parts := strings.Split(typed, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			if ss := strings.ToUpper(strings.TrimSpace(p)); ss != "" {
				out = append(out, ss)
			}
		}
		return out
	default:
		return nil
	}
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
