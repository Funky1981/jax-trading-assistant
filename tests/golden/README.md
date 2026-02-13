# Golden Tests

Golden tests capture the **current behavior** of the system as a baseline. Any future changes are validated against these golden snapshots to ensure no unintended regressions.

## Purpose

Before migrating to the modular monolith architecture, we capture:
1. **Signal generation outputs** - What signals are generated for given market conditions
2. **Trade execution decisions** - Position sizes, order types, risk calculations
3. **Orchestration flows** - AI orchestration requests and responses

## Usage

### Capturing Baseline

```powershell
# Start all services
docker-compose up -d

# Wait for services to be healthy
Start-Sleep -Seconds 10

# Capture golden baseline
go run ./tests/golden/capture.go
```

This creates timestamped snapshots in:
- `signals/baseline-YYYY-MM-DD.json`
- `executions/baseline-YYYY-MM-DD.json`
- `orchestration/baseline-YYYY-MM-DD.json`

### Validating Against Baseline

```powershell
# Run golden tests
go test -v ./tests/golden/... -tags=golden

# Or use comparison script
./scripts/compare-golden-outputs.ps1
```

### When Golden Tests Fail

**Expected Failures** (intentional changes):
1. Review the diff
2. Verify the change is intentional
3. Update the golden file: `go run ./tests/golden/cmd/capture.go`
4. Commit the new golden file with explanation

**Unexpected Failures** (regression):
1. Review the diff to understand what changed
2. Fix the regression
3. Re-run tests to verify match

## File Structure

```
tests/golden/
├── README.md              # This file
├── capture.go             # Capture tool
├── compare.go             # Comparison utilities
├── golden_test.go         # Golden test suite
├── signals/
│   ├── baseline-2026-02-13.json
│   └── ...
├── executions/
│   ├── baseline-2026-02-13.json
│   └── ...
└── orchestration/
    ├── baseline-2026-02-13.json
    └── ...
```

## Best Practices

1. **Capture before making changes** - Always have a clean baseline
2. **One golden per major feature** - Don't capture during active development
3. **Review diffs carefully** - Understand why behavior changed
4. **Version control golden files** - Commit them to git
5. **Update deliberately** - Only when change is verified correct

## Determinism

Golden tests require deterministic behavior:
- Fixed timestamps (use `testing.Clock`)
- Fixed random seeds
- Stable map iteration order
- No external API calls (use mocks)

See `../replay/` for deterministic replay testing.
