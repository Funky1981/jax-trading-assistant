# Replay Harness

The Replay Harness enables deterministic testing by replaying captured market data fixtures. This allows testing of trading strategies with known market conditions to ensure consistent behavior.

## Purpose

- **Determinism**: Same inputs always produce same outputs
- **Regression Testing**: Detect unintended changes in strategy behavior
- **Scenario Testing**: Test strategies against specific market conditions (rallies, crashes, consolidations)
- **Performance Validation**: Verify strategy performance metrics remain consistent

## Creating Fixtures

Fixtures are JSON files containing:
- Market data snapshots (prices, volumes, indicators)
- Portfolio state (cash, positions)
- Metadata (name, timestamp, description)

### Fixture Structure

```json
{
  "name": "aapl-rally",
  "description": "AAPL bullish trend scenario",
  "timestamp": "2026-02-13T09:30:00Z",
  "market_data": {
    "AAPL": {
      "price": 185.50,
      "volume": 1234567,
      "bid": 185.45,
      "ask": 185.55,
      "indicators": {
        "rsi": 65.0,
        "sma20": 183.0,
        "sma50": 180.0,
        "sma200": 175.0,
        "atr": 2.5
      }
    }
  },
  "portfolio": {
    "cash": 100000.0,
    "positions": []
  }
}
```

### Creating a New Fixture

1. Capture real market data or manually create scenario
2. Save as JSON in `tests/replay/fixtures/`
3. Ensure all required fields are present
4. Validate JSON syntax

Example:
```bash
# Create fixture from live data (future implementation)
go run ./tests/replay/tools/capture-fixture.go --symbol AAPL --output fixtures/aapl-new.json

# Or manually create JSON file
cat > fixtures/custom-scenario.json <<EOF
{
  "name": "custom-scenario",
  ...
}
EOF
```

## Running Replay Tests

### Run All Replay Tests

```bash
go test ./tests/replay/ -v
```

### Run Specific Test

```bash
go test ./tests/replay/ -v -run TestLoadFixture
```

### Verify Determinism

```bash
# Run determinism check (10 iterations by default)
go test ./tests/replay/ -v -run TestDeterminism
```

## Determinism Guarantees

The replay harness guarantees determinism by:

1. **Fixed Timestamps**: All time-based operations use fixture timestamps, not system time
2. **Isolated State**: Each replay runs in isolation with fresh state
3. **Deterministic RNG**: If randomness is needed, it uses seeded pseudo-random generators
4. **No External I/O**: Replays don't make network calls or read from external systems
5. **Sorted Maps**: Iteration over maps uses sorted keys to maintain order

### Determinism Validation

The `VerifyDeterminism` function runs a strategy multiple times (default: 10) with the same fixture and compares:
- Signal types (buy/sell/hold)
- Price levels (entry, stop loss, take profit)
- Confidence scores
- Execution timing

If any run produces different results, the test fails with detailed diff output.

## Using Replay Harness in Tests

```go
package mytest

import (
    "context"
    "testing"
    "jax-trading/tests/replay"
)

func TestMyStrategy(t *testing.T) {
    // Load fixture
    fixture, err := replay.LoadFixture("aapl-rally")
    if err != nil {
        t.Fatalf("Failed to load fixture: %v", err)
    }
    
    // Define strategy function
    strategyFunc := func(ctx context.Context, input AnalysisInput) (Signal, error) {
        strategy := NewMyStrategy()
        return strategy.Analyze(ctx, input)
    }
    
    // Run replay
    result, err := replay.ReplayStrategy(context.Background(), strategyFunc, fixture)
    if err != nil {
        t.Fatalf("Replay failed: %v", err)
    }
    
    // Verify results
    if result.Signal.Type != SignalBuy {
        t.Errorf("Expected buy signal, got %s", result.Signal.Type)
    }
}
```

## Available Fixtures

- **aapl-rally.json**: AAPL bullish trend with strong momentum
- **msft-consolidation.json**: MSFT sideways market with low volatility
- **tsla-volatility.json**: TSLA high volatility scenario

## Best Practices

1. **Version Control Fixtures**: Commit fixtures to git for team-wide consistency
2. **Descriptive Names**: Use clear, scenario-based names (not dates)
3. **Document Scenarios**: Add descriptions explaining market conditions
4. **Test Edge Cases**: Create fixtures for extreme scenarios (flash crashes, halts)
5. **Regular Updates**: Periodically update fixtures to reflect current market structure
6. **Determinism First**: Always verify determinism before using fixture in CI

## Troubleshooting

### Fixture Won't Load
- Check JSON syntax with `jq` or JSON validator
- Ensure all required fields are present
- Verify file path is relative to `tests/replay/fixtures/`

### Non-Deterministic Results
- Check for `time.Now()` calls (should use context clock)
- Look for uninitialized maps (use sorted iteration)
- Verify no external I/O (network, disk reads)
- Check for goroutines with race conditions

### Performance Issues
- Large fixtures can slow tests; split into smaller scenarios
- Use `go test -short` to skip long-running determinism checks
- Profile with `go test -cpuprofile` to find bottlenecks
