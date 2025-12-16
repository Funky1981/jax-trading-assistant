
# 05 – Agent0 Lab (Strategy Experiment Engine)

Goal: build a **separate Go process** that acts as a strategy lab, loosely
inspired by Agent0 ideas (planner + executor + reporter).  
This lab will:

- Define and run backtest experiments.
- Explore parameter grids for strategies.
- Persist the best configs as JSON.
- Generate human‑readable reports in markdown.

The lab does **not** trade or talk to the broker provider.

---

## 1. Process Layout

Add a new entrypoint:

```text
cmd/
  jax-lab/
    main.go
```

Core code lives in:

```text
internal/
  lab/
    tasks.go
    planner.go
    executor.go
    reporter.go
```

The lab will reuse:

- UTCP client (`internal/infra/utcp`).
- Backtest service.
- Storage service (optionally).

---

## 2. Core Types

### 2.1 LabTask

Defined in `internal/lab/tasks.go`:

```go
type LabTask struct {
    ID           string             `json:"id"`
    Description  string             `json:"description"`
    StrategyBase string             `json:"strategyBase"` // e.g. "earnings_gap"
    ParamGrid    map[string][]float64 `json:"paramGrid"`  // paramName -> possible values
    Symbols      []string           `json:"symbols"`
    From         time.Time          `json:"from"`
    To           time.Time          `json:"to"`
}
```

### 2.2 BacktestRunSummary

```go
type BacktestRunSummary struct {
    RunID       string   `json:"runId"`
    StrategyID  string   `json:"strategyId"`
    WinRate     float64  `json:"winRate"`
    AvgR        float64  `json:"avgR"`
    MaxDrawdown float64  `json:"maxDrawdown"`
    Sharpe      float64  `json:"sharpe"`
}
```

### 2.3 LabResult

```go
type LabResult struct {
    Task      LabTask              `json:"task"`
    Runs      []BacktestRunSummary `json:"runs"`
    Best      BacktestRunSummary   `json:"best"`
}
```

---

## 3. Planner

The planner is responsible for **creating LabTasks**.  
Initially this can be purely code‑driven (no LLM needed).

File: `internal/lab/planner.go`

Example:

```go
type Planner struct {}

func NewPlanner() *Planner { return &Planner{} }

func (p *Planner) DefaultEarningsGapTask() LabTask {
    return LabTask{
        ID:           "earnings_gap_tech_2020_2025",
        Description:  "Optimise earnings gap strategy for large‑cap tech 2020‑2025.",
        StrategyBase: "earnings_gap",
        ParamGrid: map[string][]float64{
            "gapThresholdPct": {3, 4, 5},
            "minSurprisePct":  {5, 10, 15},
            "stopAtrMult":     {1.0, 1.5, 2.0},
            "targetR":         {1.5, 2.0, 2.5},
        },
        Symbols: []string{"AAPL", "MSFT", "NVDA", "GOOGL"},
        From:    time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
        To:      time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
    }
}
```

Later, an LLM can generate such LabTasks from plain‑English prompts, but this
is optional.

---

## 4. Executor

File: `internal/lab/executor.go`

Responsibilities:

- Take a `LabTask` and generate parameter combinations.
- For each combination:
  - Build a derived `StrategyConfig` ID, e.g.
    `"earnings_gap_tech_gap3_surprise10_stop1.5_target2.0"`.
  - Call `backtest.run_strategy` via UTCP with that strategy ID + task symbols.
  - Extract stats into `BacktestRunSummary`.

Pseudo‑code:

```go
type Executor struct {
    backtest *utcp.BacktestService
}

func (e *Executor) RunTask(ctx context.Context, task LabTask) (LabResult, error) {
    var summaries []BacktestRunSummary
    for each paramCombination in expandGrid(task.ParamGrid) {
        strategyID := buildStrategyID(task.StrategyBase, paramCombination)

        stats, err := e.backtest.RunStrategy(ctx, strategyID, task.Symbols, task.From, task.To)
        if err != nil { continue }

        summaries = append(summaries, BacktestRunSummary{
            RunID:      stats.RunID,
            StrategyID: strategyID,
            WinRate:    stats.WinRate,
            AvgR:       stats.AvgR,
            MaxDrawdown: stats.MaxDrawdown,
            Sharpe:      stats.Sharpe,
        })
    }

    best := pickBestConfig(summaries)
    return LabResult{Task: task, Runs: summaries, Best: best}, nil
}
```

`expandGrid` and `pickBestConfig` should be simple, deterministic helpers.

Objective example:

- Maximise `Sharpe`, subject to `MaxDrawdown >= -0.2` and `WinRate >= 0.45`.

---

## 5. Reporter

File: `internal/lab/reporter.go`

Responsibilities:

- Turn `LabResult` into markdown reports.
- Save them under e.g. `reports/{taskID}.md`.

Example structure:

```md
# Lab Report – earnings_gap_tech_2020_2025

## Objective
Optimise earnings gap strategy for large‑cap tech (2020‑2025).

## Best Configuration
- Strategy ID: earnings_gap_tech_gap3_surprise10_stop1.5_target2.0
- Win rate: 57%
- Avg R: 1.4
- Max drawdown: -18%
- Sharpe: 1.7

## Top 5 Configurations
| Strategy ID | WinRate | AvgR | MaxDD | Sharpe |
|------------|---------|------|-------|--------|
| ...        | ...     | ...  | ...   | ...    |

## Notes
- Tighter stops (<1.0 ATR) caused worse drawdowns.
- Performance concentrated in NVDA and AVGO.
```

Reporter steps:

1. Create directories if they don’t exist (`internal/reports/`).  
2. Render markdown from templates or string builders.  
3. Optionally also dump JSON of `LabResult` for machine consumption.

---

## 6. Output Strategy Configs

For the **best configuration**, the lab should:

1. Produce a concrete `StrategyConfig` JSON file.  
2. Save under `config/strategies/`, e.g.:

   `config/strategies/earnings_gap_tech_v3.json`

The file format should match the `StrategyConfig` struct from Jax Core. The
derived parameters (thresholds, ATR multipliers, etc.) can be embedded into
free‑form rule strings or additional numeric fields (extend the struct as
needed).

Jax Core will load all `*.json` strategies at startup and can be instructed
via config which ones are “active”.

---

## 7. cmd/jax-lab/main.go

The entrypoint should:

1. Load config (if any).  
2. Initialise UTCP client + BacktestService.  
3. Instantiate Planner, Executor, Reporter.  
4. For now, run one default task and exit:

```go
func main() {
    ctx := context.Background()

    planner := lab.NewPlanner()
    task := planner.DefaultEarningsGapTask()

    exec := lab.NewExecutor(backtestService)
    result, err := exec.RunTask(ctx, task)
    if err != nil { log.Fatal(err) }

    reporter := lab.NewReporter("internal/reports")
    if err := reporter.WriteReport(result); err != nil { log.Fatal(err) }

    if err := lab.WriteBestStrategyConfig("config/strategies", result); err != nil {
        log.Fatal(err)
    }
}
```

Later you can add:

- CLI flags (choose task, date ranges, etc.).  
- HTTP interface or scheduler.

---

## 8. Tasks for Codex / AI

1. Create `cmd/jax-lab/main.go` and wire up UTCP client + services.
2. Implement `LabTask`, `BacktestRunSummary`, `LabResult` structs.
3. Implement `Planner` with at least one hard‑coded example task.
4. Implement `Executor` with:
   - `expandGrid` helper for parameter grids.
   - `pickBestConfig` helper based on Sharpe + drawdown constraints.
5. Implement `Reporter` to generate markdown reports under `internal/reports/`.
6. Implement helper `WriteBestStrategyConfig` to dump best config as JSON
   under `config/strategies/`.
7. Test the lab by running a fake or stubbed `backtest.run_strategy` tool that
   returns dummy stats so the flow can be validated.
