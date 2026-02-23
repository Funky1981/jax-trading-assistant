# Research Projects and Evaluation

## Objective
Make research repeatable, comparable, and hard to fool yourself with.

## Research project structure
Add:
- `research/projects/<project_id>/project.json`
- `research/projects/<project_id>/runs/`

Example `project.json`:
- `instanceId`
- `symbols`
- `from`, `to`
- `parameterGrid`
- `fillModel`

## Required features
### 1) Parameter sweeps
- grid execution of parameters
- record every run with config hash
- rank runs by:
  - max drawdown (primary)
  - average daily P/L
  - tail loss

### 2) Walk-forward evaluation (minimum)
- split date range:
  - in-sample (tune)
  - out-of-sample (validate)
- store metrics separately

### 3) Metrics (minimum)
- total P/L
- max drawdown
- win rate
- avg win / avg loss
- worst day
- % days traded
- exposure (% of session in position)

## Acceptance criteria
- Re-running project yields same results.
- Runs are traceable to:
  - instance config hash
  - fill model config
  - git commit hash (if available)
