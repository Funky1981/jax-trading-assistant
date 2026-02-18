# Phase 2 Validation Scripts

This directory contains scripts to validate Phase 2 of the ADR-0012 migration: in-process signal generation.

## Overview

Phase 2 replaces the HTTP call from the API to `jax-signal-generator` with an in-process call in `cmd/trader`. These scripts validate that the behavior is identical.

## Scripts

### `capture-baseline.ps1`

Captures golden baseline from the current `jax-signal-generator` service.

**Usage**:
```powershell
.\tests\phase2\capture-baseline.ps1
```

**What it does**:
1. Calls `jax-signal-generator` API with test symbols
2. Saves signal output to `tests/phase2/golden/signals-baseline.json`
3. Creates timestamp metadata file

### `compare-outputs.ps1`

Compares `cmd/trader` output against golden baseline.

**Usage**:
```powershell
.\tests\phase2\compare-outputs.ps1
```

**What it does**:
1. Calls `cmd/trader` API with same test symbols
2. Saves output to `tests/phase2/output/signals-trader.json`
3. Compares with golden baseline
4. Reports differences (should be ZERO for phase completion)

**Exit codes**:
- `0`: Outputs match (success)
- `1`: Outputs differ (validation failed)

### `run-validation.ps1`

End-to-end validation workflow.

**Usage**:
```powershell
.\tests\phase2\run-validation.ps1
```

**What it does**:
1. Verifies both services are running
2. Captures baseline from `jax-signal-generator`
3. Generates signals from `cmd/trader`
4. Compares outputs
5. Generates validation report

## Validation Criteria

### Signal Equality

For Phase 2 to pass, the following must be **identical** between `jax-signal-generator` and `cmd/trader`:

- Signal type (BUY/SELL)
- Symbol
- Strategy ID
- Confidence (±0.01)
- Entry price (±0.01)
- Stop loss (±0.01)
- Take profit levels (±0.01)
- Technical indicators (RSI, MACD, etc.)

### Acceptable Differences

- Signal ID (UUIDs are unique)
- Timestamp (within 1 second tolerance)
- Processing duration

## Directory Structure

```
tests/phase2/
├── README.md                    # This file
├── capture-baseline.ps1         # Capture golden baseline
├── compare-outputs.ps1          # Compare trader vs baseline
├── run-validation.ps1           # End-to-end validation
├── golden/                      # Golden baseline outputs
│   └── signals-baseline.json
├── output/                      # cmd/trader outputs
│   └── signals-trader.json
└── reports/                     # Validation reports
    └── validation-YYYYMMDD-HHMMSS.txt
```

## Example Workflow

```powershell
# 1. Start services
docker-compose up -d jax-signal-generator postgres

# 2. Wait for services to be healthy
Start-Sleep -Seconds 10

# 3. Capture baseline
.\tests\phase2\capture-baseline.ps1

# 4. Build and start cmd/trader
go build -o trader.exe ./cmd/trader
$env:DATABASE_URL="postgresql://jax:jax@localhost:5433/jax"
$env:PORT="8100"
Start-Process -FilePath ".\trader.exe" -NoNewWindow

# 5. Wait for trader startup
Start-Sleep -Seconds 5

# 6. Run validation
.\tests\phase2\run-validation.ps1

# 7. Check results
# Exit code 0 = success, 1 = failure
```

## Troubleshooting

### Services Not Running

**Error**: `Failed to connect to http://localhost:8096`

**Solution**:
```powershell
docker-compose ps
docker-compose up -d jax-signal-generator
```

### No Market Data

**Error**: `Generated 0 signals`

**Solution**: Ensure market data is loaded:
```powershell
docker-compose exec postgres psql -U jax -d jax -c "SELECT COUNT(*) FROM candles;"
```

If no candles exist, run market data service first:
```powershell
docker-compose up -d jax-market
```

### Golden Baseline Missing

**Error**: `Golden baseline not found`

**Solution**:
```powershell
.\tests\phase2\capture-baseline.ps1
```

## Golden Test Philosophy

Golden tests lock in current behavior before refactoring. They ensure:

1. **No behavior drift**: Refactored code produces identical outputs
2. **Regression detection**: Any change is immediately visible
3. **Confidence**: Safe to merge when golden tests pass

For Phase 2, passing golden tests means signal generation is **provably identical** between the HTTP service and in-process implementation.

## References

- [ADR-0012: Modular Monolith](../../Docs/ADR-0012-two-runtime-modular-monolith.md)
- [Phase 2 Documentation](../../Docs/PHASE_2_COMPLETE.md)
- [cmd/trader README](../../cmd/trader/README.md)
