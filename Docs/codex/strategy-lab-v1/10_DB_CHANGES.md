# DB Changes (Files + DB storage)

## Objective
Support instance isolation and persistent research runs.

## Existing tables observed
- `events`, `trades`, `audit_events`
- `strategy_signals` used by trade executor (queried directly)

## Required schema changes (minimum)

### 1) strategy_instances
Table: `strategy_instances`
- `instance_id` TEXT PRIMARY KEY
- `strategy_id` TEXT NOT NULL
- `enabled` BOOLEAN NOT NULL DEFAULT true
- `config_json` JSONB NOT NULL
- `config_hash` TEXT NOT NULL
- `created_at` TIMESTAMPTZ NOT NULL DEFAULT now()
- `updated_at` TIMESTAMPTZ NOT NULL DEFAULT now()

Index:
- `(enabled)`
- `(strategy_id)`

### 2) strategy_signals tagging
Add to `strategy_signals`:
- `instance_id` TEXT NOT NULL
- index `(instance_id, created_at desc)`

### 3) trades tagging
Add to `trades`:
- `instance_id` TEXT NULL (backfill for old rows)
- index `(instance_id, created_at desc)`

### 4) backtest runs
Table: `backtest_runs`
- `run_id` TEXT PRIMARY KEY
- `instance_id` TEXT NOT NULL REFERENCES strategy_instances(instance_id)
- `strategy_id` TEXT NOT NULL
- `from_ts` TIMESTAMPTZ NOT NULL
- `to_ts` TIMESTAMPTZ NOT NULL
- `config_hash` TEXT NOT NULL
- `fill_model_json` JSONB NOT NULL
- `stats_json` JSONB NOT NULL
- `created_at` TIMESTAMPTZ NOT NULL DEFAULT now()

Table: `backtest_trades`
- `id` TEXT PRIMARY KEY
- `run_id` TEXT NOT NULL REFERENCES backtest_runs(run_id)
- `symbol` TEXT NOT NULL
- `entry_ts` TIMESTAMPTZ NOT NULL
- `exit_ts` TIMESTAMPTZ NOT NULL
- `side` TEXT NOT NULL
- `entry_price` DOUBLE PRECISION NOT NULL
- `exit_price` DOUBLE PRECISION NOT NULL
- `qty` INT NOT NULL
- `pnl` DOUBLE PRECISION NOT NULL
- `r_multiple` DOUBLE PRECISION NULL

Indexes:
- `(run_id)`
- `(symbol, entry_ts)`

## Acceptance criteria
- Every signal/trade/run is queryable by `instance_id`.
- Runs are reproducible using stored config + fill model.
