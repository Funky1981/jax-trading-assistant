# Phase 1 Testing Guide

## Prerequisites
1. Docker and Docker Compose installed
2. PostgreSQL running (via docker-compose)
3. IB Bridge configured (optional for testing, can run without live data)

## Testing Steps

### 1. Build and Start Services

```powershell
# Navigate to project root
cd "c:\Projects\jax-trading assistant"

# Bring down any existing services
docker compose down

# Build and start all services (including new jax-market)
docker compose up --build -d

# Check service status
docker compose ps
```

Expected output: All services should be "healthy" or "running"

### 2. Verify Database Migration

```powershell
# Connect to PostgreSQL container
docker compose exec postgres psql -U jax -d jax

# Check if new tables exist
\dt

# Verify strategy_signals table structure
\d strategy_signals

# Verify orchestration_runs table structure
\d orchestration_runs

# Verify trade_approvals table structure
\d trade_approvals

# Exit psql
\q
```

Expected: All three new tables should exist with proper schema

### 3. Test jax-market Health Endpoint

```powershell
# Check health
curl http://localhost:8095/health

# Expected response:
# {"status":"healthy","service":"jax-market","uptime":"10.5s"}
```

### 4. Test jax-market Metrics Endpoint

```powershell
# Check metrics
curl http://localhost:8095/metrics

# Expected response (example):
# {
#   "total_ingestions": 5,
#   "successful_ingests": 48,
#   "failed_ingests": 2,
#   "last_ingest_time": "2026-02-05T14:30:00Z",
#   "last_ingest_duration": "2.5s",
#   "symbol_count": 10,
#   "uptime": "5m30s"
# }
```

### 5. Check Market Data Ingestion

```powershell
# Connect to database
docker compose exec postgres psql -U jax -d jax

# Check if quotes are being ingested
SELECT symbol, price, timestamp, updated_at FROM quotes ORDER BY updated_at DESC LIMIT 10;

# Check if candles are being stored
SELECT symbol, timestamp, close, volume FROM candles ORDER BY timestamp DESC LIMIT 10;

# Exit
\q
```

Expected: Data should be appearing for symbols in watchlist (AAPL, MSFT, GOOGL, etc.)

### 6. Monitor Service Logs

```powershell
# View jax-market logs
docker compose logs -f jax-market

# Look for:
# - "market data client initialized with 10 symbol(s)"
# - "jax-market started (interval: 60s)"
# - "starting ingestion for 10 symbols"
# - "ingestion complete: 10 success, 0 errors in 2.5s"
# - "ingested AAPL: price=$175.43, 30 candles"
```

### 7. Test Service Dependencies

```powershell
# Verify jax-market can connect to postgres
docker compose logs jax-market | Select-String "database connected"

# Verify jax-market can connect to IB Bridge (if enabled)
docker compose logs jax-market | Select-String "IB"

# Check all service health
curl http://localhost:8888/health  # hindsight
curl http://localhost:8090/health  # jax-memory (might be /tools POST)
curl http://localhost:8092/health  # ib-bridge
curl http://localhost:8093/health  # agent0-service
curl http://localhost:8095/health  # jax-market (NEW!)
```

## Troubleshooting

### Issue: jax-market won't start

```powershell
# Check logs
docker compose logs jax-market

# Common issues:
# 1. Database not ready - wait 30s and check again
# 2. IB Bridge not accessible - set ib.enabled=false in config
# 3. Build failed - check go.mod dependencies
```

### Issue: No data in quotes/candles tables

```powershell
# Check IB Bridge status
curl http://localhost:8092/health

# Check jax-market config
docker compose exec jax-market cat /app/config/jax-market.json

# Verify IB Bridge is enabled and reachable
# If IB Gateway is not running, data won't ingest
# For testing without IB, you can enable Polygon/Alpaca providers
```

### Issue: Migration didn't run

```powershell
# Manually run migration
docker compose exec postgres psql -U jax -d jax -f /docker-entrypoint-initdb.d/000004_signals_and_runs.up.sql

# OR copy migration file and run
docker cp db/postgres/migrations/000004_signals_and_runs.up.sql jax-trading-assistant-postgres-1:/tmp/
docker compose exec postgres psql -U jax -d jax -f /tmp/000004_signals_and_runs.up.sql
```

## Success Criteria ✅

- [x] jax-market service starts successfully
- [x] Health endpoint returns 200 OK
- [x] Metrics endpoint shows ingestion statistics
- [x] Database tables created (strategy_signals, orchestration_runs, trade_approvals)
- [x] Market data appearing in quotes table
- [x] Candles data appearing in candles table
- [x] No critical errors in logs
- [x] Service runs continuously (check after 5 minutes)

## Next Steps (Phase 2)

Once Phase 1 is verified working:
1. Implement signal generation service
2. Create signal API endpoints in jax-api
3. Build background signal generator
4. Test signal detection and storage

## Configuration Notes

### Default Watchlist
The initial watchlist contains:
- AAPL, MSFT, GOOGL, AMZN (big tech)
- TSLA, META, NVDA, AMD (growth stocks)
- NFLX (streaming)
- SPY (S&P 500 ETF)

To modify: Edit `config/jax-market.json`

### Ingestion Interval
Default: 60 seconds (1 minute)

To change:
```json
{
  "ingest_interval": 300  // 5 minutes
}
```

### Data Providers
Default: IB Bridge only

To enable Polygon.io:
```json
{
  "polygon": {
    "enabled": true,
    "api_key": "YOUR_API_KEY",
    "tier": "free"
  }
}
```

## Cleanup

```powershell
# Stop all services
docker compose down

# Remove volumes (WARNING: deletes all data)
docker compose down -v
```
