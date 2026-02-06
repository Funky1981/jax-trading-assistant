# Phase 1 Troubleshooting Guide

## Common Issues & Solutions

### Issue 1: "docker compose up -d --build" takes forever or fails

**Symptoms:**
- Build process hangs or takes >10 minutes
- Gets stuck downloading Go modules or installing dependencies
- Build errors about missing files or modules

**Solutions:**

#### Option A: Build only what you need (Recommended for Phase 1)
```powershell
# Use the Phase 1 quick start script
.\scripts\start-phase1.ps1
```

#### Option B: Build services individually
```powershell
# Start postgres first
docker compose up -d postgres
Start-Sleep -Seconds 10

# Build and start jax-market only
docker compose up -d --build jax-market
```

#### Option C: Check Docker resources
```powershell
# Make sure Docker has enough resources:
# Docker Desktop -> Settings -> Resources
# - CPUs: At least 4
# - Memory: At least 8GB
# - Disk: At least 20GB free
```

---

### Issue 2: "curl http://localhost:8095/health" returns connection refused

**Symptoms:**
```
curl: (7) Failed to connect to localhost port 8095 after X ms: Connection refused
```

**Diagnosis:**
```powershell
# Check if service is running
docker compose ps jax-market

# Check service logs
docker compose logs jax-market
```

**Solutions:**

1. **Service not started:**
   ```powershell
   docker compose up -d jax-market
   ```

2. **Service crashed:**
   ```powershell
   # Check logs for errors
   docker compose logs --tail=50 jax-market
   
   # Common errors:
   # - "database connection failed" -> postgres not ready
   # - "IB Bridge connection failed" -> normal, service continues anyway
   ```

3. **Still starting up:**
   ```powershell
   # Wait 30 seconds and try again
   Start-Sleep -Seconds 30
   curl http://localhost:8095/health
   ```

---

### Issue 3: Go build fails with "requires go >= 1.24.0"

**Symptoms:**
```
go: go.mod requires go >= 1.24.0 (running go 1.22.X)
```

**Solution:**
This was fixed in the Dockerfile. Rebuild:
```powershell
docker compose build jax-market --no-cache
docker compose up -d jax-market
```

---

### Issue 4: No data appearing in database

**Symptoms:**
```sql
SELECT * FROM quotes;
-- Returns 0 rows
```

**Diagnosis:**
```powershell
# Check if jax-market is running
docker compose ps jax-market

# Check metrics
curl http://localhost:8095/metrics | ConvertFrom-Json

# Look for:
# - total_ingestions > 0
# - failed_ingests (should be low)
# - last_ingest_time (should be recent)
```

**Solutions:**

1. **IB Bridge not connected:**
   ```powershell
   # Check IB Bridge status
   docker compose ps ib-bridge
   curl http://localhost:8092/health
   
   # If unhealthy, restart it:
   docker compose restart ib-bridge
   ```

2. **Service just started:**
   ```powershell
   # Wait for first ingestion cycle (default: 60 seconds)
   Start-Sleep -Seconds 70
   
   # Check database again
   docker compose exec postgres psql -U jax -d jax -c "SELECT symbol, price, updated_at FROM quotes ORDER BY updated_at DESC LIMIT 5;"
   ```

3. **Check logs for errors:**
   ```powershell
   docker compose logs --tail=100 jax-market | Select-String "error"
   ```

---

### Issue 5: Database migration didn't run

**Symptoms:**
```sql
ERROR:  relation "strategy_signals" does not exist
```

**Solution:**
```powershell
# Manually run migration
docker compose exec postgres psql -U jax -d jax

# In psql:
\i /docker-entrypoint-initdb.d/000004_signals_and_runs.up.sql
\q
```

Or copy the file and run it:
```powershell
docker cp "db/postgres/migrations/000004_signals_and_runs.up.sql" jax-trading-assistant-postgres-1:/tmp/
docker compose exec postgres psql -U jax -d jax -f /tmp/000004_signals_and_runs.up.sql
```

---

### Issue 6: Port 8095 already in use

**Symptoms:**
```
Error bind: address already in use
```

**Solution:**
```powershell
# Find what's using port 8095
Get-NetTCPConnection -LocalPort 8095 | Select-Object -Property OwningProcess | ForEach-Object { Get-Process -Id $_.OwningProcess }

# Kill the process or change port in docker-compose.yml:
# ports:
#   - "8096:8095"  # Use 8096 externally instead
```

---

### Issue 7: "out of memory" during build

**Symptoms:**
```
ERROR: failed to solve: process exited with code 137
```

**Solution:**
```powershell
# Increase Docker memory limit
# Docker Desktop -> Settings -> Resources -> Memory: 8GB minimum

# Or build without cache:
docker compose build jax-market --no-cache --memory=4g
```

---

## Quick Diagnostic Commands

### Check all service health
```powershell
docker compose ps
```

### View realtime logs
```powershell
docker compose logs -f jax-market
```

### Test database connection
```powershell
docker compose exec postgres psql -U jax -d jax -c "\dt"
```

### Check disk space
```powershell
docker system df
```

### Clean up if needed
```powershell
# Remove stopped containers
docker compose down

# Remove volumes (WARNING: deletes data!)
docker compose down -v

# Remove build cache
docker builder prune -a
```

---

## Emergency Reset

If nothing works, nuclear option:
```powershell
# 1. Stop everything
docker compose down -v

# 2. Remove all images (WARNING: forces complete rebuild)
docker compose down --rmi all

# 3. Clean Docker system
docker system prune -a --volumes

# 4. Start fresh
.\scripts\start-phase1.ps1
```

---

## Getting Help

If issues persist:

1. **Collect diagnostic info:**
   ```powershell
   # Save to file
   docker compose logs jax-market > jax-market-logs.txt
   docker compose ps > services-status.txt
   docker system df > docker-disk.txt
   ```

2. **Check specific error messages**
   - Search logs for "ERROR", "FATAL", "panic"
   - Look for stack traces

3. **Verify prerequisites:**
   - Docker Desktop running
   - At least 8GB RAM available
   - At least 20GB disk space
   - PostgreSQL port 5432 not in use
   - jax-market port 8095 not in use

---

## Success Indicators

When Phase 1 is working correctly, you should see:

✅ **Service Status:**
```
docker compose ps jax-market
# STATUS: Up X minutes (healthy)
```

✅ **Health Check:**
```powershell
curl http://localhost:8095/health
# {"status":"healthy","service":"jax-market","uptime":"5m30s"}
```

✅ **Metrics:**
```powershell
curl http://localhost:8095/metrics
# total_ingestions > 0, successful_ingests > 0
```

✅ **Database:**
```sql
SELECT COUNT(*) FROM quotes;
# Should return > 0 after 60+ seconds
```

✅ **Logs:**
```
docker compose logs --tail=10 jax-market
# Should show: "ingestion complete: N success, 0 errors"
```

If all checks pass, Phase 1 is operational! 🎉
