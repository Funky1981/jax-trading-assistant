# Jax Trader Runtime

**Status**: Phase 2 Implementation (In-Process Signal Generation)  
**Version**: 0.1.0  
**ADR**: [ADR-0012 Modular Monolith Migration](../../Docs/ADR-0012-two-runtime-modular-monolith.md)

## Overview

`cmd/trader` is the production trader runtime that consolidates trading operations into a single process. This represents Phase 2 of the ADR-0012 migration toward a modular monolith architecture.

### What It Does

- **Signal Generation**: In-process execution of trading strategies (RSI Momentum, MACD Crossover, MA Crossover)
- **Market Data Integration**: Real-time technical indicator calculation from database candles
- **Database Persistence**: Stores generated signals in PostgreSQL
- **HTTP API**: Compatible with existing `jax-signal-generator` endpoints

### Phase 2 Goals

1. ✅ Replace HTTP call to `jax-signal-generator` with in-process function call
2. ✅ Preserve 100% identical signal generation behavior
3. ✅ Maintain API compatibility for gradual migration
4. ⏳ Validate with golden tests and replay harness

## Architecture

### Before (Phase 1)
```
jax-api → HTTP → jax-signal-generator → Postgres
```

### After (Phase 2)
```
cmd/trader (in-process) → Postgres
    ├─ Signal Generator (internal/trader/signalgenerator)
    ├─ Strategy Registry (libs/strategies)
    └─ Contracts Layer (libs/contracts)
```

## Usage

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | `postgresql://jax:jax@localhost:5432/jax` | PostgreSQL connection string |
| `PORT` | `8100` | HTTP server port |

### Running Locally

```bash
# Set environment
export DATABASE_URL="postgresql://jax:jax@localhost:5432/jax"
export PORT=8100

# Build and run
go build -o trader ./cmd/trader
./trader
```

### Running with Docker

```bash
docker build -f cmd/trader/Dockerfile -t jax-trader:latest .

docker run -d \
  --name jax-trader \
  -p 8100:8100 \
  -e DATABASE_URL="postgresql://jax:jax@postgres:5432/jax" \
  jax-trader:latest
```

### Health Check

```bash
curl http://localhost:8100/health
```

Expected response:
```json
{
  "service": "jax-trader",
  "version": "0.1.0",
  "status": "healthy",
  "uptime": "5m30s"
}
```

## API Endpoints

### POST /api/v1/signals/generate

Generate trading signals for specified symbols.

**Request**:
```json
{
  "symbols": ["AAPL", "MSFT", "GOOGL"]
}
```

**Response**:
```json
{
  "success": true,
  "signals": [
    {
      "id": "uuid",
      "symbol": "AAPL",
      "type": "BUY",
      "confidence": 0.85,
      "entry_price": 150.25,
      "stop_loss": 148.00,
      "take_profit": [152.50, 154.75],
      "reason": "RSI oversold at 28.5, bullish reversal expected",
      "strategy_id": "rsi_momentum_v1",
      "timestamp": "2026-02-13T10:30:00Z"
    }
  ],
  "count": 1,
  "duration": "234ms"
}
```

### GET /api/v1/signals?symbol=AAPL&limit=50

Retrieve signal history for a symbol.

**Query Parameters**:
- `symbol` (required): Stock symbol
- `limit` (optional): Max results (default: 50, max: 500)

**Response**:
```json
{
  "symbol": "AAPL",
  "signals": [...],
  "count": 10
}
```

### GET /metrics

Retrieve service metrics.

## Implementation Details

### In-Process Signal Generator

Location: `internal/trader/signalgenerator/inprocess.go`

**Key Features**:
- Implements `services.SignalGenerator` interface from `libs/contracts`
- Uses strategy registry from `libs/strategies`
- Calculates technical indicators (RSI, MACD, SMA, ATR, Bollinger Bands)
- Stores signals in `strategy_signals` table
- Zero HTTP overhead compared to microservice architecture

**Strategies Registered**:
1. RSI Momentum (`rsi_momentum_v1`)
2. MACD Crossover (`macd_crossover_v1`)
3. MA Crossover (`ma_crossover_v1`)

### Database Schema

Uses existing tables:
- `strategy_signals`: Signal storage with metadata
- `candles`: Historical OHLCV data for indicator calculation
- `quotes`: Real-time price quotes

No schema changes required - full compatibility with Phase 1.

## Testing

### Unit Tests

```bash
go test ./internal/trader/signalgenerator/... -v
```

Expected output:
```
PASS: TestNew
PASS: TestCalculateRSI
PASS: TestCalculateSMA
PASS: TestCalculateATR
PASS: TestCalculateBollingerBands
PASS: TestDetermineTrend
PASS: TestHealthCheck
PASS: TestCalculateAvgVolume
```

### Integration Test

```bash
# Start trader
./trader

# Generate signals
curl -X POST http://localhost:8100/api/v1/signals/generate \
  -H "Content-Type: application/json" \
  -d '{"symbols": ["AAPL"]}'

# Verify signals stored
curl "http://localhost:8100/api/v1/signals?symbol=AAPL&limit=10"
```

## Migration Path

### Parallel Validation (Current)

Both `jax-signal-generator` (port 8096) and `cmd/trader` (port 8100) run simultaneously:

1. `jax-signal-generator` generates signals via HTTP
2. `cmd/trader` generates signals in-process
3. Golden tests compare outputs for equality
4. Replay harness validates determinism

### Cutover (Future)

Once validation passes:

1. Update `jax-api` to call `cmd/trader` instead of `jax-signal-generator`
2. Stop `jax-signal-generator` service
3. Remove from `docker-compose.yml`

## Performance Characteristics

### Expected Improvements

- **Latency**: ~50ms reduction (no HTTP round-trip)
- **Memory**: Shared connection pool, registry
- **Reliability**: No network failures between services

### Benchmarks

```bash
go test -bench=. ./internal/trader/signalgenerator/
```

Expected results:
- `BenchmarkCalculateRSI`: ~5-10 µs per operation
- `BenchmarkCalculateSMA`: ~2-5 µs per operation
- `BenchmarkDetermineTrend`: ~100 ns per operation

## Troubleshooting

### Database Connection Fails

**Symptom**: `failed to ping database` on startup

**Solution**:
```bash
# Verify DATABASE_URL is correct
echo $DATABASE_URL

# Test PostgreSQL connectivity
psql $DATABASE_URL -c "SELECT 1"

# Check database is running
docker ps | grep postgres
```

### No Strategies Registered

**Symptom**: `no strategies registered` in health check

**Solution**: This should not occur - strategies are hard-coded in `main.go`. If it does, check build configuration.

### Signal Generation Returns Empty

**Symptom**: `POST /api/v1/signals/generate` returns `count: 0`

**Possible Causes**:
- No market data in database (run `jax-market` first)
- Symbols have insufficient candle history (need 50+ candles)
- All signals below confidence threshold (0.6)

**Debug**:
```sql
-- Check candle availability
SELECT symbol, COUNT(*) 
FROM candles 
GROUP BY symbol;

-- Check recent quotes
SELECT symbol, price, timestamp 
FROM quotes 
ORDER BY timestamp DESC 
LIMIT 10;
```

## Next Steps (Phase 3)

1. Collapse orchestration HTTP seam (in-process orchestrator)
2. Integrate trade executor call path
3. Add artifact approval workflow
4. Introduce `cmd/research` for strategy development

## References

- [ADR-0012: Modular Monolith Architecture](../../Docs/ADR-0012-two-runtime-modular-monolith.md)
- [Phase 1 Complete](../../Docs/PHASE_1_COMPLETE.md)
- [Service Contracts](../../libs/contracts/README.md)
- [Strategy Library](../../libs/strategies/README.md)
